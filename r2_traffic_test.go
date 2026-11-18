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

func r2GaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		t.Fatal(err)
	}
	return metric.GetGauge().GetValue()
}

func TestR2TrafficCollection(t *testing.T) {
	var mode atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables struct {
				AccountID string    `json:"accountID"`
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
		if variables.AccountID != "test-account" || variables.MaxTime.Sub(variables.MinTime) != r2TrafficWindow || variables.MaxTime.Second() != 0 {
			t.Errorf("incorrect analytics variables: %+v", variables)
		}
		w.Header().Set("Content-Type", "application/json")
		switch mode.Load() {
		case 1:
			fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{"bandwidth":[],"operations":[]}]}}}`)
		case 2:
			fmt.Fprint(w, `{"errors":[{"message":"denied"}]}`)
		default:
			fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{
                "bandwidth":[{"dimensions":{"bucketName":"eu_example-bucket"},"sum":{"bytesUpload":6000,"bytesDownload":9000}}],
                "operations":[
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
		for _, metric := range r2TrafficMetrics {
			metric.Reset()
		}
	})
	account := cfaccounts.Account{ID: "test-account", Name: "Example account"}
	buckets := []string{"eu_example-bucket", "idle-bucket"}
	fetchR2TrafficForAccount(account, buckets)
	for _, check := range []struct {
		name  string
		gauge prometheus.Gauge
		want  float64
	}{
		{"upload", r2Upload.WithLabelValues(account.Name, buckets[0]), 20},
		{"download", r2Download.WithLabelValues(account.Name, buckets[0]), 30},
		{"all errors", r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "all"), .12},
		{"4xx errors", r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "4xx"), .07},
		{"5xx errors", r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "5xx"), .03},
		{"other errors", r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "other"), .02},
		{"throttling", r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "429"), .02},
		{"idle errors", r2ErrorRatio.WithLabelValues(account.Name, buckets[1], "all"), 0},
	} {
		if got := r2GaugeValue(t, check.gauge); got != check.want {
			t.Errorf("%s: got %v, want %v", check.name, got, check.want)
		}
	}
	if r2GaugeValue(t, r2TrafficUpdated.WithLabelValues(account.Name)) == 0 {
		t.Fatal("successful collection must set freshness")
	}
	mode.Store(1)
	fetchR2TrafficForAccount(account, buckets)
	if r2GaugeValue(t, r2Upload.WithLabelValues(account.Name, buckets[0])) != 0 || r2GaugeValue(t, r2ErrorRatio.WithLabelValues(account.Name, buckets[0], "all")) != 0 {
		t.Fatal("successful empty window must clear old measurements to zero")
	}
	r2Upload.WithLabelValues("Other account", "other-bucket").Set(123)
	mode.Store(2)
	fetchR2TrafficForAccount(account, buckets)
	if n := r2Upload.DeletePartialMatch(prometheus.Labels{"account": account.Name}); n != 0 {
		t.Fatal("failed collection retained old throughput")
	}
	if r2GaugeValue(t, r2TrafficUpdated.WithLabelValues(account.Name)) != 0 {
		t.Fatal("failed collection must invalidate freshness")
	}
	if r2GaugeValue(t, r2Upload.WithLabelValues("Other account", "other-bucket")) != 123 {
		t.Fatal("failure affected another account")
	}
}
