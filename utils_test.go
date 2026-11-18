package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	"github.com/machinebox/graphql"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", "/"},
		{"/health", "/health"},
		{"/api/v1/users", "/api/v1/users"},
		{"/users/123", "/users/:id"},
		{"/users/123/orders/456", "/users/:id/orders/:id"},
		{"/orders/550e8400-e29b-41d4-a716-446655440000", "/orders/:uuid"},
		{"/items/5f3a2b1c9d", "/items/:id"},
		{"/search?q=test&page=1", "/search"},
		{"/api/v1/users/123?include=orders", "/api/v1/users/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func r2GaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		t.Fatal(err)
	}
	return metric.GetGauge().GetValue()
}

func TestR2MetricsCollection(t *testing.T) {
	var mode atomic.Int32
	var calls atomic.Int32
	const daily = `"r2StorageAdaptiveGroups":[
        {"dimensions":{"bucketName":"eu_example-bucket"},"max":{"payloadSize":12345}},
        {"dimensions":{"bucketName":"idle-bucket"},"max":{"payloadSize":0}}
    ],"r2OperationsAdaptiveGroups":[
        {"dimensions":{"bucketName":"eu_example-bucket","actionType":"GetObject"},"sum":{"requests":1234}}
    ]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Variables struct {
				AccountID string    `json:"accountID"`
				Date      string    `json:"date"`
				MinTime   time.Time `json:"mintime"`
				MaxTime   time.Time `json:"maxtime"`
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			http.Error(w, "invalid query", http.StatusBadRequest)
			return
		}
		variables := body.Variables
		if variables.AccountID != "test-account" || variables.Date != variables.MaxTime.Format("2006-01-02") || variables.MaxTime.Sub(variables.MinTime) != r2TrafficWindow || variables.MaxTime.Second() != 0 {
			t.Errorf("incorrect analytics variables: %+v", variables)
		}
		w.Header().Set("Content-Type", "application/json")
		switch mode.Load() {
		case 1:
			fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{`+daily+`,"r2BandwidthUsageAdaptiveGroups":[],"r2TrafficOperations":[]}]}}}`)
		case 2:
			fmt.Fprint(w, `{"errors":[{"message":"denied"}]}`)
		default:
			fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{`+daily+`,
                "r2BandwidthUsageAdaptiveGroups":[{"dimensions":{"bucketName":"eu_example-bucket"},"sum":{"bytesUpload":6000,"bytesDownload":9000}}],
                "r2TrafficOperations":[
                    {"dimensions":{"bucketName":"eu_example-bucket","actionStatus":"success","responseStatusCode":200},"sum":{"requests":88}},
                    {"dimensions":{"bucketName":"eu_example-bucket","actionStatus":"userError","responseStatusCode":404},"sum":{"requests":5}},
                    {"dimensions":{"bucketName":"eu_example-bucket","actionStatus":"userError","responseStatusCode":429},"sum":{"requests":2}},
                    {"dimensions":{"bucketName":"eu_example-bucket","actionStatus":"internalError","responseStatusCode":503},"sum":{"requests":3}},
                    {"dimensions":{"bucketName":"eu_example-bucket","actionStatus":"internalError","responseStatusCode":0},"sum":{"requests":2}}
                ]
            }]}}}`)
		}
	}))
	defer server.Close()
	oldGQL, oldTimeout := gql, cftimeout
	gql = &GraphQL{Client: graphql.NewClient(server.URL), Mu: &sync.RWMutex{}}
	cftimeout = time.Second
	t.Cleanup(func() {
		gql, cftimeout = oldGQL, oldTimeout
		r2Upload.Reset()
		r2Download.Reset()
		r2ErrorRatio.Reset()
		r2TrafficUpdated.Reset()
		r2Storage.Reset()
		r2StorageTotal.Reset()
		r2Operation.Reset()
	})
	account := cfaccounts.Account{ID: "test-account", Name: "Example account"}
	buckets := []string{"eu_example-bucket", "idle-bucket"}
	collect := func() {
		t.Helper()
		before := calls.Load()
		var wg sync.WaitGroup
		wg.Add(1)
		fetchR2StorageForAccount(account, &wg)
		wg.Wait()
		if got := calls.Load() - before; got != 1 {
			t.Errorf("got %d R2 API calls, want one", got)
		}
	}
	collect()
	for _, check := range []struct {
		name  string
		gauge prometheus.Gauge
		want  float64
	}{
		{"daily storage", r2Storage.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0]}), 12345},
		{"daily operations", r2Operation.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "operation": "GetObject"}), 1234},
		{"upload", r2Upload.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0]}), 20},
		{"download", r2Download.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0]}), 30},
		{"all errors", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "all"}), .12},
		{"4xx errors", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "4xx"}), .07},
		{"5xx errors", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "5xx"}), .03},
		{"other errors", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "other"}), .02},
		{"throttling", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "429"}), .02},
		{"idle errors", r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[1], "status_class": "all"}), 0},
	} {
		t.Run(check.name, func(t *testing.T) {
			if got := r2GaugeValue(t, check.gauge); got != check.want {
				t.Errorf("got %v, want %v", got, check.want)
			}
		})
	}
	if r2GaugeValue(t, r2TrafficUpdated.With(prometheus.Labels{"account": account.Name})) == 0 {
		t.Fatal("successful collection must set freshness")
	}
	mode.Store(1)
	collect()
	if r2GaugeValue(t, r2Upload.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0]})) != 0 || r2GaugeValue(t, r2ErrorRatio.With(prometheus.Labels{"account": account.Name, "bucket": buckets[0], "status_class": "all"})) != 0 {
		t.Fatal("successful empty window must clear old measurements to zero")
	}
	r2Upload.With(prometheus.Labels{"account": "Other account", "bucket": "other-bucket"}).Set(123)
	mode.Store(2)
	collect()
	if n := r2Upload.DeletePartialMatch(prometheus.Labels{"account": account.Name}); n != 0 {
		t.Fatal("failed collection retained old throughput")
	}
	if r2GaugeValue(t, r2TrafficUpdated.With(prometheus.Labels{"account": account.Name})) != 0 {
		t.Fatal("failed collection must invalidate freshness")
	}
	if r2GaugeValue(t, r2Upload.With(prometheus.Labels{"account": "Other account", "bucket": "other-bucket"})) != 123 {
		t.Fatal("failure affected another account")
	}
}
