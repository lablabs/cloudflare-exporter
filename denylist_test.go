package main

import (
	"sync"
	"testing"

	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/spf13/viper"
)

// Regression: cloudflare_r2_storage_bytes could be registered but was missing
// from the known-metrics set, so denylisting it aborted startup.
func TestBuildFilteredMetricsSetAcceptsR2StorageBytes(t *testing.T) {
	if _, err := buildFilteredMetricsSet([]string{r2StorageMetricName.String()}); err != nil {
		t.Errorf("buildFilteredMetricsSet(%q): %v", r2StorageMetricName, err)
	}
	if _, err := buildFilteredMetricsSet([]string{"cloudflare_not_a_metric"}); err == nil {
		t.Error("expected an error for an unknown metric name")
	}
}

// The Cloudflare clients are nil under test, so any fetcher that reaches the
// API panics. Denying every metric a fetcher feeds must make it return first.
func TestDeniedFetchersSkipAPICalls(t *testing.T) {
	orig := deniedMetrics
	deniedMetrics = MetricsSet{}
	for mn := range allMetrics {
		deniedMetrics.Add(mn)
	}
	t.Cleanup(func() { deniedMetrics = orig })

	// These two fetchers are also behind feature flags that default to off, so
	// enable them or the denylist guard is not what makes them return.
	viper.Set("enable_edge_errors_by_path", true)
	viper.Set("enable_account_usage_metrics", true)
	t.Cleanup(func() {
		viper.Set("enable_edge_errors_by_path", false)
		viper.Set("enable_account_usage_metrics", false)
	})

	account := cfaccounts.Account{ID: "account-id", Name: "account"}
	zones := []cfzones.Zone{{ID: "zone-id", Name: "zone"}}

	var wg sync.WaitGroup
	for _, f := range []func(cfaccounts.Account, *sync.WaitGroup){
		fetchLoadblancerPoolsHealth,
		fetchWorkerAnalytics,
		fetchLogpushAnalyticsForAccount,
		fetchR2StorageForAccount,
		fetchZeroTrustAnalyticsForAccount,
		fetchAccountHTTPDataTransferAnalytics,
	} {
		wg.Add(1)
		f(account, &wg)
	}
	for _, f := range []func([]cfzones.Zone, *sync.WaitGroup){
		fetchZoneColocationAnalytics,
		fetchLoadBalancerAnalytics,
		fetchLogpushAnalyticsForZone,
		fetchZoneASNAnalytics,
		fetchEdgeErrorsByPathAnalytics,
	} {
		wg.Add(1)
		f(zones, &wg)
	}
	wg.Wait()
}
