package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	cf "github.com/cloudflare/cloudflare-go/v4"
	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	cfoption "github.com/cloudflare/cloudflare-go/v4/option"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/viper"
)

type asnGroup struct {
	asn   string
	count uint64
	bytes uint64
}

// Anonymous nested structs in zoneRespASN are awkward to build by hand, so go through JSON.
func newZoneRespASN(t *testing.T, groups ...asnGroup) *zoneRespASN {
	t.Helper()

	items := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		items = append(items, map[string]any{
			"count": g.count,
			"dimensions": map[string]any{
				"clientAsn":            g.asn,
				"clientASNDescription": "desc-" + g.asn,
			},
			"sum": map[string]any{"edgeResponseBytes": g.bytes},
		})
	}

	raw, err := json.Marshal(map[string]any{"zoneTag": "tag", "httpRequestsAdaptiveGroups": items})
	if err != nil {
		t.Fatal(err)
	}
	var z zoneRespASN
	if err := json.Unmarshal(raw, &z); err != nil {
		t.Fatal(err)
	}
	return &z
}

func asnLabels(zone, asn string) prometheus.Labels {
	return prometheus.Labels{"zone": zone, "account": "acc", "asn": asn, "asn_description": "desc-" + asn}
}

func TestAddASNGroupsClearsStaleSeries(t *testing.T) {
	zoneRequestASN.Reset()
	zoneBandwidthASN.Reset()

	// Scrape 1: zone-a sees ASN 100 and 200, zone-b sees ASN 300.
	addASNGroups(newZoneRespASN(t, asnGroup{"100", 10, 1000}, asnGroup{"200", 20, 2000}), "zone-a", "acc")
	addASNGroups(newZoneRespASN(t, asnGroup{"300", 30, 3000}), "zone-b", "acc")

	if got := testutil.CollectAndCount(zoneRequestASN); got != 3 {
		t.Fatalf("after scrape 1: zoneRequestASN has %d series, want 3", got)
	}

	// Scrape 2: ASN 100 disappears from zone-a, ASN 400 shows up; zone-b is not scraped.
	addASNGroups(newZoneRespASN(t, asnGroup{"200", 5, 500}, asnGroup{"400", 7, 700}), "zone-a", "acc")

	for _, vec := range []*prometheus.CounterVec{zoneRequestASN, zoneBandwidthASN} {
		if got := testutil.CollectAndCount(vec); got != 3 {
			t.Errorf("after scrape 2: %d series, want 3 (zone-a/200, zone-a/400, zone-b/300)", got)
		}
	}

	// Values must reflect the latest scrape only, not accumulate across scrapes.
	checks := []struct {
		vec   *prometheus.CounterVec
		zone  string
		asn   string
		value float64
	}{
		{zoneRequestASN, "zone-a", "200", 5},
		{zoneRequestASN, "zone-a", "400", 7},
		{zoneRequestASN, "zone-b", "300", 30},
		{zoneBandwidthASN, "zone-a", "200", 500},
		{zoneBandwidthASN, "zone-a", "400", 700},
		{zoneBandwidthASN, "zone-b", "300", 3000},
	}
	for _, c := range checks {
		if got := testutil.ToFloat64(c.vec.With(asnLabels(c.zone, c.asn))); got != c.value {
			t.Errorf("%s/%s = %v, want %v", c.zone, c.asn, got, c.value)
		}
	}
}

func TestAddASNGroupsSeriesCountStaysBounded(t *testing.T) {
	zoneRequestASN.Reset()
	zoneBandwidthASN.Reset()

	// Every scrape brings one never-seen-before ASN; the number of series must not grow with it.
	for i := 0; i < 100; i++ {
		addASNGroups(newZoneRespASN(t, asnGroup{"13335", 1, 1}, asnGroup{fmt.Sprintf("%d", 60000+i), 1, 1}), "zone-a", "acc")
	}

	for _, vec := range []*prometheus.CounterVec{zoneRequestASN, zoneBandwidthASN} {
		if got := testutil.CollectAndCount(vec); got != 2 {
			t.Errorf("after 100 scrapes: %d series, want 2", got)
		}
	}
}

func TestAddHTTPGroupsClearsStaleThreatTypes(t *testing.T) {
	zoneThreatsType.Reset()

	newZoneResp := func(threatTypes ...string) *zoneResp {
		t.Helper()
		pathing := make([]map[string]any, 0, len(threatTypes))
		for _, name := range threatTypes {
			pathing = append(pathing, map[string]any{"threatPathingName": name, "requests": 1})
		}
		raw, err := json.Marshal(map[string]any{
			"httpRequests1mGroups": []map[string]any{{"sum": map[string]any{"threatPathingMap": pathing}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		var z zoneResp
		if err := json.Unmarshal(raw, &z); err != nil {
			t.Fatal(err)
		}
		return &z
	}

	addHTTPGroups(newZoneResp("bic.ban.unknown", "hot.ban.unknown"), "zone-a", "acc")
	addHTTPGroups(newZoneResp("hot.ban.unknown"), "zone-a", "acc")

	if got := testutil.CollectAndCount(zoneThreatsType); got != 1 {
		t.Errorf("zoneThreatsType has %d series, want 1", got)
	}
}

func TestAddASNGroupsEmptyResponseClearsSeries(t *testing.T) {
	zoneRequestASN.Reset()
	zoneBandwidthASN.Reset()

	addASNGroups(newZoneRespASN(t, asnGroup{"100", 10, 100}), "zone", "acc")
	addASNGroups(newZoneRespASN(t), "zone", "acc")

	for _, vec := range []*prometheus.CounterVec{zoneRequestASN, zoneBandwidthASN} {
		if n := testutil.CollectAndCount(vec); n != 0 {
			t.Fatalf("empty ASN response retained %d series", n)
		}
	}
}

// mockTransport serves canned Cloudflare REST and GraphQL responses.
type mockTransport func(*http.Request) (int, string)

func (f mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	status, body := f(r)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func mockCloudflareAPI(t *testing.T, handler mockTransport) {
	t.Helper()

	oldCF, oldGQL, oldTimeout := cfclient, gql, cftimeout
	oldFree := viper.GetBool("free_tier")

	client := &http.Client{Transport: handler}
	cfclient = cf.NewClient(cfoption.WithHTTPClient(client), cfoption.WithMaxRetries(0))
	gql = NewGraphQLClient(client)
	cftimeout = time.Second
	viper.Set("free_tier", false)

	t.Cleanup(func() {
		cfclient, gql, cftimeout = oldCF, oldGQL, oldTimeout
		viper.Set("free_tier", oldFree)
	})
}

func runFetch(fetch func(*sync.WaitGroup)) {
	var wg sync.WaitGroup
	wg.Add(1)
	fetch(&wg)
	wg.Wait()
}

func TestFetchZoneColocationAnalyticsEmptyResponseClearsSeries(t *testing.T) {
	mockCloudflareAPI(t, func(*http.Request) (int, string) {
		return 200, `{"data":{"viewer":{"zones":[{"zoneTag":"zone","httpRequestsAdaptiveGroups":[]}]}}}`
	})

	vectors := []*prometheus.CounterVec{zoneColocationVisits, zoneColocationEdgeResponseBytes, zoneColocationRequestsTotal}
	for _, vec := range vectors {
		vec.Reset()
		vec.WithLabelValues("zone", "", "colo", "host").Add(1)
	}

	runFetch(func(wg *sync.WaitGroup) {
		fetchZoneColocationAnalytics([]cfzones.Zone{{ID: "zone", Name: "zone"}}, wg)
	})

	for _, vec := range vectors {
		if n := testutil.CollectAndCount(vec); n != 0 {
			t.Fatalf("empty colocation response retained %d series", n)
		}
	}
}

func TestFetchLogpushAnalyticsForZoneKeepsSnapshotOnBatchFailure(t *testing.T) {
	calls := 0
	fail := true
	mockCloudflareAPI(t, func(*http.Request) (int, string) {
		calls++
		if fail && calls == 2 {
			return 500, `{"errors":[{"message":"unavailable"}]}`
		}
		return 200, `{"data":{"viewer":{"zones":[{"logpushHealthAdaptiveGroups":[{"count":3,"dimensions":{"jobId":2,"destinationType":"s3","final":1}}]}]}}}`
	})

	logpushFailedJobsZone.Reset()
	old := prometheus.Labels{"destination": "s3", "job_id": "1", "final": "1"}
	logpushFailedJobsZone.With(old).Add(7)

	// More zones than one GraphQL request allows, so the fetch runs two batches.
	zones := make([]cfzones.Zone, cfgraphqlreqlimit+1)
	for i := range zones {
		zones[i].ID = fmt.Sprint(i)
	}
	fetch := func() {
		runFetch(func(wg *sync.WaitGroup) { fetchLogpushAnalyticsForZone(zones, wg) })
	}

	fetch()
	if n := testutil.CollectAndCount(logpushFailedJobsZone); n != 1 {
		t.Fatalf("failed batch changed series count: %d", n)
	}
	if v := testutil.ToFloat64(logpushFailedJobsZone.With(old)); v != 7 {
		t.Fatalf("failed batch changed old value: %v", v)
	}

	fail = false
	calls = 0
	fetch()
	if n := testutil.CollectAndCount(logpushFailedJobsZone); n != 1 {
		t.Fatalf("successful batches retained stale series: %d", n)
	}
	if v := testutil.ToFloat64(logpushFailedJobsZone.WithLabelValues("s3", "2", "1")); v != 6 {
		t.Fatalf("batch total: got %v, want 6", v)
	}
}

func TestAddCloudflareTunnelStatusSnapshots(t *testing.T) {
	mode := "empty"
	mockCloudflareAPI(t, func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "connections") {
			if mode == "connector-error" {
				return 500, `{"success":false,"errors":[{"code":1000,"message":"unavailable"}]}`
			}
			return 200, `{"success":true,"result":[]}`
		}
		if mode == "list-error" {
			return 500, `{"success":false,"errors":[{"code":1000,"message":"unavailable"}]}`
		}
		if mode == "empty" || r.URL.Query().Get("page") == "2" {
			return 200, `{"success":true,"result":[],"result_info":{"page":1,"total_pages":1}}`
		}
		return 200, `{"success":true,"result":[{"id":"tunnel","name":"name","status":"healthy","tun_type":"cfd_tunnel"}],"result_info":{"page":1,"total_pages":1}}`
	})

	account := cfaccounts.Account{ID: "account", Name: "acc"}
	for _, scenario := range []string{"empty", "list-error", "connector-error", "empty-connectors"} {
		t.Run(scenario, func(t *testing.T) {
			mode = scenario
			tunnelInfo.Reset()
			tunnelHealthStatus.Reset()
			tunnelConnectorInfo.Reset()
			tunnelConnectorActiveConnections.Reset()

			// Previous snapshot: one tunnel with one connector.
			tunnelInfo.WithLabelValues("acc", "tunnel", "name", "cfd_tunnel").Set(1)
			tunnelHealthStatus.WithLabelValues("acc", "tunnel").Set(0)
			tunnelConnectorInfo.WithLabelValues("acc", "tunnel", "client", "v1", "amd64", "ip").Set(1)
			tunnelConnectorActiveConnections.WithLabelValues("acc", "tunnel", "client").Set(4)

			addCloudflareTunnelStatus(account)

			// On any error the previous snapshot must survive untouched.
			wantConnectors := 0
			if strings.Contains(scenario, "error") {
				wantConnectors = 1
			}
			for _, vec := range []*prometheus.GaugeVec{tunnelConnectorInfo, tunnelConnectorActiveConnections} {
				if n := testutil.CollectAndCount(vec); n != wantConnectors {
					t.Fatalf("connector series: got %d, want %d", n, wantConnectors)
				}
			}
			if scenario == "empty" {
				if n := testutil.CollectAndCount(tunnelInfo); n != 0 {
					t.Fatal("deleted tunnel retained")
				}
			}
			if strings.Contains(scenario, "error") {
				if v := testutil.ToFloat64(tunnelHealthStatus.WithLabelValues("acc", "tunnel")); v != 0 {
					t.Fatal("failed fetch changed health")
				}
			}
		})
	}
}
