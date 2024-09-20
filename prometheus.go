package main

import (
	"strconv"
	"strings"
	"sync"

	"github.com/biter777/countries"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type MetricName string

func (mn MetricName) String() string {
	return string(mn)
}

const (
	zoneRequestTotalMetricName                   MetricName = "cloudflare_zone_requests_total"
	zoneRequestCachedMetricName                  MetricName = "cloudflare_zone_requests_cached"
	zoneRequestSSLEncryptedMetricName            MetricName = "cloudflare_zone_requests_ssl_encrypted"
	zoneRequestContentTypeMetricName             MetricName = "cloudflare_zone_requests_content_type"
	zoneRequestCountryMetricName                 MetricName = "cloudflare_zone_requests_country"
	zoneRequestHTTPStatusMetricName              MetricName = "cloudflare_zone_requests_status"
	zoneRequestBrowserMapMetricName              MetricName = "cloudflare_zone_requests_browser_map_page_views_count"
	zoneRequestOriginStatusCountryHostMetricName MetricName = "cloudflare_zone_requests_origin_status_country_host"
	zoneRequestStatusCountryHostMetricName       MetricName = "cloudflare_zone_requests_status_country_host"
	zoneBandwidthTotalMetricName                 MetricName = "cloudflare_zone_bandwidth_total"
	zoneBandwidthCachedMetricName                MetricName = "cloudflare_zone_bandwidth_cached"
	zoneBandwidthSSLEncryptedMetricName          MetricName = "cloudflare_zone_bandwidth_ssl_encrypted"
	zoneBandwidthContentTypeMetricName           MetricName = "cloudflare_zone_bandwidth_content_type"
	zoneBandwidthCountryMetricName               MetricName = "cloudflare_zone_bandwidth_country"
	zoneThreatsTotalMetricName                   MetricName = "cloudflare_zone_threats_total"
	zoneThreatsCountryMetricName                 MetricName = "cloudflare_zone_threats_country"
	zoneThreatsTypeMetricName                    MetricName = "cloudflare_zone_threats_type"
	zonePageviewsTotalMetricName                 MetricName = "cloudflare_zone_pageviews_total"
	zoneUniquesTotalMetricName                   MetricName = "cloudflare_zone_uniques_total"
	zoneColocationVisitsMetricName               MetricName = "cloudflare_zone_colocation_visits"
	zoneColocationEdgeResponseBytesMetricName    MetricName = "cloudflare_zone_colocation_edge_response_bytes"
	zoneColocationRequestsTotalMetricName        MetricName = "cloudflare_zone_colocation_requests_total"
	zoneFirewallEventsCountMetricName            MetricName = "cloudflare_zone_firewall_events_count"
	zoneHealthCheckEventsOriginCountMetricName   MetricName = "cloudflare_zone_health_check_events_origin_count"
	workerRequestsMetricName                     MetricName = "cloudflare_worker_requests_count"
	workerErrorsMetricName                       MetricName = "cloudflare_worker_errors_count"
	workerCPUTimeMetricName                      MetricName = "cloudflare_worker_cpu_time"
	workerDurationMetricName                     MetricName = "cloudflare_worker_duration"
	poolHealthStatusMetricName                   MetricName = "cloudflare_zone_pool_health_status"
	poolRequestsTotalMetricName                  MetricName = "cloudflare_zone_pool_requests_total"
	logpushFailedJobsAccountMetricName           MetricName = "cloudflare_logpush_failed_jobs_account_count"
	logpushFailedJobsZoneMetricName              MetricName = "cloudflare_logpush_failed_jobs_zone_count"
)

type MetricsSet map[MetricName]struct{}

func (ms MetricsSet) Has(mn MetricName) bool {
	_, exists := ms[mn]
	return exists
}

func (ms MetricsSet) Add(mn MetricName) {
	ms[mn] = struct{}{}
}

var (
	counterMetrics = map[MetricName]*prometheus.CounterVec{
		// Requests
		zoneRequestTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestTotalMetricName.String(),
			Help: "Number of requests for zone",
		}, []string{"zone", "account"},
		),

		zoneRequestCachedMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestCachedMetricName.String(),
			Help: "Number of cached requests for zone",
		}, []string{"zone", "account"},
		),

		zoneRequestSSLEncryptedMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestSSLEncryptedMetricName.String(),
			Help: "Number of encrypted requests for zone",
		}, []string{"zone", "account"},
		),

		zoneRequestContentTypeMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestContentTypeMetricName.String(),
			Help: "Number of request for zone per content type",
		}, []string{"zone", "account", "content_type"},
		),

		zoneRequestCountryMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestCountryMetricName.String(),
			Help: "Number of request for zone per country",
		}, []string{"zone", "account", "country", "region"},
		),

		zoneRequestHTTPStatusMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestHTTPStatusMetricName.String(),
			Help: "Number of request for zone per HTTP status",
		}, []string{"zone", "account", "status"},
		),

		zoneRequestBrowserMapMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestBrowserMapMetricName.String(),
			Help: "Number of successful requests for HTML pages per zone",
		}, []string{"zone", "account", "family"},
		),

		zoneRequestOriginStatusCountryHostMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestOriginStatusCountryHostMetricName.String(),
			Help: "Count of not cached requests for zone per origin HTTP status per country per host",
		}, []string{"zone", "account", "status", "country", "host"},
		),
		zoneRequestStatusCountryHostMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneRequestStatusCountryHostMetricName.String(),
			Help: "Count of requests for zone per edge HTTP status per country per host",
		}, []string{"zone", "account", "status", "country", "host"},
		),

		zoneBandwidthTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneBandwidthTotalMetricName.String(),
			Help: "Total bandwidth per zone in bytes",
		}, []string{"zone", "account"},
		),

		zoneBandwidthCachedMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneBandwidthCachedMetricName.String(),
			Help: "Cached bandwidth per zone in bytes",
		}, []string{"zone", "account"},
		),

		zoneBandwidthSSLEncryptedMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneBandwidthSSLEncryptedMetricName.String(),
			Help: "Encrypted bandwidth per zone in bytes",
		}, []string{"zone", "account"},
		),

		zoneBandwidthContentTypeMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneBandwidthContentTypeMetricName.String(),
			Help: "Bandwidth per zone per content type",
		}, []string{"zone", "account", "content_type"},
		),

		zoneBandwidthCountryMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneBandwidthCountryMetricName.String(),
			Help: "Bandwidth per country per zone",
		}, []string{"zone", "account", "country", "region"},
		),

		zoneThreatsTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneThreatsTotalMetricName.String(),
			Help: "Threats per zone",
		}, []string{"zone", "account"},
		),

		zoneThreatsCountryMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneThreatsCountryMetricName.String(),
			Help: "Threats per zone per country",
		}, []string{"zone", "account", "country", "region"},
		),

		zoneThreatsTypeMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneThreatsTypeMetricName.String(),
			Help: "Threats per zone per type",
		}, []string{"zone", "account", "type"},
		),

		zonePageviewsTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zonePageviewsTotalMetricName.String(),
			Help: "Pageviews per zone",
		}, []string{"zone", "account"},
		),

		zoneUniquesTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneUniquesTotalMetricName.String(),
			Help: "Uniques per zone",
		}, []string{"zone", "account"},
		),

		zoneColocationVisitsMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneColocationVisitsMetricName.String(),
			Help: "Total visits per colocation",
		}, []string{"zone", "account", "colocation", "host"},
		),

		zoneColocationEdgeResponseBytesMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneColocationEdgeResponseBytesMetricName.String(),
			Help: "Edge response bytes per colocation",
		}, []string{"zone", "account", "colocation", "host"},
		),

		zoneColocationRequestsTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneColocationRequestsTotalMetricName.String(),
			Help: "Total requests per colocation",
		}, []string{"zone", "account", "colocation", "host"},
		),

		zoneFirewallEventsCountMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneFirewallEventsCountMetricName.String(),
			Help: "Count of Firewall events",
		}, []string{"zone", "account", "action", "source", "rule", "host", "country"},
		),

		zoneHealthCheckEventsOriginCountMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: zoneHealthCheckEventsOriginCountMetricName.String(),
			Help: "Number of Heath check events per region per origin",
		}, []string{"zone", "account", "health_status", "origin_ip", "region", "fqdn"},
		),

		workerRequestsMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: workerRequestsMetricName.String(),
			Help: "Number of requests sent to worker by script name",
		}, []string{"script_name", "account"},
		),

		workerErrorsMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: workerErrorsMetricName.String(),
			Help: "Number of errors by script name",
		}, []string{"script_name", "account"},
		),

		poolRequestsTotalMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: poolRequestsTotalMetricName.String(),
			Help: "Requests per pool",
		},
			[]string{"zone", "account", "load_balancer_name", "pool_name", "origin_name"},
		),
		// TODO: Update this to counter vec and use counts from the query to add
		logpushFailedJobsAccountMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: logpushFailedJobsAccountMetricName.String(),
			Help: "Number of failed logpush jobs on the account level",
		},
			[]string{"account", "destination", "job_id", "final"},
		),

		logpushFailedJobsZoneMetricName: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: logpushFailedJobsZoneMetricName.String(),
			Help: "Number of failed logpush jobs on the zone level",
		},
			[]string{"destination", "job_id", "final"},
		),
	}

	gaugeMetrics = map[MetricName]*prometheus.GaugeVec{
		workerCPUTimeMetricName: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: workerCPUTimeMetricName.String(),
			Help: "CPU time quantiles by script name",
		}, []string{"script_name", "account", "quantile"},
		),

		workerDurationMetricName: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: workerDurationMetricName.String(),
			Help: "Duration quantiles by script name (GB*s)",
		}, []string{"script_name", "account", "quantile"},
		),

		poolHealthStatusMetricName: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: poolHealthStatusMetricName.String(),
			Help: "Reports the health of a pool, 1 for healthy, 0 for unhealthy.",
		},
			[]string{"zone", "account", "load_balancer_name", "pool_name"},
		),
	}
)

func disableDeniedMetrics(metricsDenylist []string) {
	for _, metric := range metricsDenylist {
		delete(counterMetrics, MetricName(metric))
		delete(gaugeMetrics, MetricName(metric))
	}
}

func mustRegisterMetrics() {
	for _, metric := range counterMetrics {
		prometheus.MustRegister(metric)
	}

	for _, metric := range gaugeMetrics {
		prometheus.MustRegister(metric)
	}
}

func fetchWorkerAnalytics(account cloudflare.Account, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	r, err := fetchWorkerTotals(account.ID)
	if err != nil {
		return
	}

	// Replace spaces with hyphens and convert to lowercase
	accountName := strings.ToLower(strings.ReplaceAll(account.Name, " ", "-"))

	for _, a := range r.Viewer.Accounts {
		for _, w := range a.WorkersInvocationsAdaptive {
			if m, ok := counterMetrics[workerRequestsMetricName]; ok {
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName}).Add(float64(w.Sum.Requests))
			}
			if m, ok := counterMetrics[workerErrorsMetricName]; ok {
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName}).Add(float64(w.Sum.Errors))
			}
			if m, ok := gaugeMetrics[workerCPUTimeMetricName]; ok {
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P50"}).Set(float64(w.Quantiles.CPUTimeP50))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P75"}).Set(float64(w.Quantiles.CPUTimeP75))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P99"}).Set(float64(w.Quantiles.CPUTimeP99))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P999"}).Set(float64(w.Quantiles.CPUTimeP999))
			}
			if m, ok := gaugeMetrics[workerDurationMetricName]; ok {
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P50"}).Set(float64(w.Quantiles.DurationP50))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P75"}).Set(float64(w.Quantiles.DurationP75))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P99"}).Set(float64(w.Quantiles.DurationP99))
				m.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "quantile": "P999"}).Set(float64(w.Quantiles.DurationP999))
			}
		}
	}
}

func fetchLogpushAnalyticsForAccount(account cloudflare.Account, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	if viper.GetBool("free_tier") {
		return
	}

	r, err := fetchLogpushAccount(account.ID)

	if err != nil {
		return
	}

	for _, acc := range r.Viewer.Accounts {
		for _, LogpushHealthAdaptiveGroup := range acc.LogpushHealthAdaptiveGroups {
			if m, ok := counterMetrics[logpushFailedJobsAccountMetricName]; ok {
				m.With(prometheus.Labels{"account": account.ID,
					"destination": LogpushHealthAdaptiveGroup.Dimensions.DestinationType,
					"job_id":      strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.JobID),
					"final":       strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.Final)}).Add(float64(LogpushHealthAdaptiveGroup.Count))
			}
		}
	}
}

func fetchLogpushAnalyticsForZone(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchLogpushZone(zoneIDs)

	if err != nil {
		return
	}

	for _, zone := range r.Viewer.Zones {
		for _, LogpushHealthAdaptiveGroup := range zone.LogpushHealthAdaptiveGroups {
			if m, ok := counterMetrics[logpushFailedJobsZoneMetricName]; ok {
				m.With(prometheus.Labels{"destination": LogpushHealthAdaptiveGroup.Dimensions.DestinationType,
					"job_id": strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.JobID),
					"final":  strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.Final)}).Add(float64(LogpushHealthAdaptiveGroup.Count))
			}
		}
	}
}

func fetchZoneColocationAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// Colocation metrics are not available in non-enterprise zones
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchColoTotals(zoneIDs)
	if err != nil {
		return
	}
	for _, z := range r.Viewer.Zones {
		cg := z.ColoGroups
		name, account := findZoneAccountName(zones, z.ZoneTag)
		for _, c := range cg {
			if m, ok := counterMetrics[zoneColocationVisitsMetricName]; ok {
				m.With(prometheus.Labels{"zone": name, "account": account, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.Visits))
			}
			if m, ok := counterMetrics[zoneColocationEdgeResponseBytesMetricName]; ok {
				m.With(prometheus.Labels{"zone": name, "account": account, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.EdgeResponseBytes))
			}
			if m, ok := counterMetrics[zoneColocationRequestsTotalMetricName]; ok {
				m.With(prometheus.Labels{"zone": name, "account": account, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Count))
			}
		}
	}
}

func fetchZoneAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// None of the below referenced metrics are available in the free tier
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchZoneTotals(zoneIDs)
	if err != nil {
		return
	}

	for _, z := range r.Viewer.Zones {
		name, account := findZoneAccountName(zones, z.ZoneTag)
		z := z

		addHTTPGroups(&z, name, account)
		addFirewallGroups(&z, name, account)
		addHealthCheckGroups(&z, name, account)
		addHTTPAdaptiveGroups(&z, name, account)
	}
}

func addHTTPGroups(z *zoneResp, name string, account string) {
	// Nothing to do.
	if len(z.HTTP1mGroups) == 0 {
		return
	}

	zt := z.HTTP1mGroups[0]

	if m, ok := counterMetrics[zoneRequestTotalMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.Requests))
	}
	if m, ok := counterMetrics[zoneRequestCachedMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.CachedRequests))
	}
	if m, ok := counterMetrics[zoneRequestSSLEncryptedMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.EncryptedRequests))
	}

	for _, ct := range zt.Sum.ContentType {
		if m, ok := counterMetrics[zoneRequestContentTypeMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "content_type": ct.EdgeResponseContentType}).Add(float64(ct.Requests))
		}
		if m, ok := counterMetrics[zoneBandwidthContentTypeMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "content_type": ct.EdgeResponseContentType}).Add(float64(ct.Bytes))
		}
	}

	for _, country := range zt.Sum.Country {
		c := countries.ByName(country.ClientCountryName)
		region := c.Info().Region.Info().Name

		if m, ok := counterMetrics[zoneRequestCountryMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "country": country.ClientCountryName, "region": region}).Add(float64(country.Requests))
		}
		if m, ok := counterMetrics[zoneBandwidthCountryMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "country": country.ClientCountryName, "region": region}).Add(float64(country.Bytes))
		}
		if m, ok := counterMetrics[zoneThreatsCountryMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "country": country.ClientCountryName, "region": region}).Add(float64(country.Threats))
		}
	}

	for _, status := range zt.Sum.ResponseStatus {
		if m, ok := counterMetrics[zoneRequestHTTPStatusMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "status": strconv.Itoa(status.EdgeResponseStatus)}).Add(float64(status.Requests))
		}
	}

	for _, browser := range zt.Sum.BrowserMap {
		if m, ok := counterMetrics[zoneRequestBrowserMapMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "family": browser.UaBrowserFamily}).Add(float64(browser.PageViews))
		}
	}

	if m, ok := counterMetrics[zoneBandwidthTotalMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.Bytes))
	}
	if m, ok := counterMetrics[zoneBandwidthCachedMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.CachedBytes))
	}
	if m, ok := counterMetrics[zoneBandwidthSSLEncryptedMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.EncryptedBytes))
	}
	if m, ok := counterMetrics[zoneThreatsTotalMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.Threats))
	}

	for _, t := range zt.Sum.ThreatPathing {
		if m, ok := counterMetrics[zoneThreatsTypeMetricName]; ok {
			m.With(prometheus.Labels{"zone": name, "account": account, "type": t.Name}).Add(float64(t.Requests))
		}
	}

	if m, ok := counterMetrics[zonePageviewsTotalMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Sum.PageViews))
	}

	// Uniques
	if m, ok := counterMetrics[zoneUniquesTotalMetricName]; ok {
		m.With(prometheus.Labels{"zone": name, "account": account}).Add(float64(zt.Unique.Uniques))
	}
}

func addFirewallGroups(z *zoneResp, name string, account string) {
	// Nothing to do.
	if len(z.FirewallEventsAdaptiveGroups) == 0 {
		return
	}
	rulesMap := fetchFirewallRules(z.ZoneTag)
	for _, g := range z.FirewallEventsAdaptiveGroups {
		if m, ok := counterMetrics[zoneFirewallEventsCountMetricName]; ok {
			m.With(
				prometheus.Labels{
					"zone":    name,
					"account": account,
					"action":  g.Dimensions.Action,
					"source":  g.Dimensions.Source,
					"rule":    normalizeRuleName(rulesMap[g.Dimensions.RuleID]),
					"host":    g.Dimensions.ClientRequestHTTPHost,
					"country": g.Dimensions.ClientCountryName,
				}).Add(float64(g.Count))
		}
	}
}

func normalizeRuleName(initialText string) string {
	maxLength := 200
	nonSpaceName := strings.ReplaceAll(strings.ToLower(initialText), " ", "_")
	if len(nonSpaceName) > maxLength {
		return nonSpaceName[:maxLength]
	}
	return nonSpaceName
}

func addHealthCheckGroups(z *zoneResp, name string, account string) {
	if len(z.HealthCheckEventsAdaptiveGroups) == 0 {
		return
	}

	for _, g := range z.HealthCheckEventsAdaptiveGroups {
		if m, ok := counterMetrics[zoneHealthCheckEventsOriginCountMetricName]; ok {
			m.With(
				prometheus.Labels{
					"zone":          name,
					"account":       account,
					"health_status": g.Dimensions.HealthStatus,
					"origin_ip":     g.Dimensions.OriginIP,
					"region":        g.Dimensions.Region,
					"fqdn":          g.Dimensions.Fqdn,
				}).Add(float64(g.Count))
		}
	}
}

func addHTTPAdaptiveGroups(z *zoneResp, name string, account string) {
	for _, g := range z.HTTPRequestsAdaptiveGroups {
		if m, ok := counterMetrics[zoneRequestOriginStatusCountryHostMetricName]; ok {
			m.With(
				prometheus.Labels{
					"zone":    name,
					"account": account,
					"status":  strconv.Itoa(int(g.Dimensions.OriginResponseStatus)),
					"country": g.Dimensions.ClientCountryName,
					"host":    g.Dimensions.ClientRequestHTTPHost,
				}).Add(float64(g.Count))
		}
	}

	for _, g := range z.HTTPRequestsEdgeCountryHost {
		if m, ok := counterMetrics[zoneRequestStatusCountryHostMetricName]; ok {
			m.With(
				prometheus.Labels{
					"zone":    name,
					"account": account,
					"status":  strconv.Itoa(int(g.Dimensions.EdgeResponseStatus)),
					"country": g.Dimensions.ClientCountryName,
					"host":    g.Dimensions.ClientRequestHTTPHost,
				}).Add(float64(g.Count))
		}
	}
}

func fetchLoadBalancerAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// None of the below referenced metrics are available in the free tier
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	l, err := fetchLoadBalancerTotals(zoneIDs)
	if err != nil {
		return
	}
	for _, lb := range l.Viewer.Zones {
		name, account := findZoneAccountName(zones, lb.ZoneTag)
		lb := lb
		addLoadBalancingRequestsAdaptive(&lb, name, account)
		addLoadBalancingRequestsAdaptiveGroups(&lb, name, account)
	}
}

func addLoadBalancingRequestsAdaptiveGroups(z *lbResp, name string, account string) {
	for _, g := range z.LoadBalancingRequestsAdaptiveGroups {
		if m, ok := counterMetrics[poolRequestsTotalMetricName]; ok {
			m.With(
				prometheus.Labels{
					"zone":               name,
					"account":            account,
					"load_balancer_name": g.Dimensions.LbName,
					"pool_name":          g.Dimensions.SelectedPoolName,
					"origin_name":        g.Dimensions.SelectedOriginName,
				}).Add(float64(g.Count))
		}
	}
}

func addLoadBalancingRequestsAdaptive(z *lbResp, name string, account string) {
	for _, g := range z.LoadBalancingRequestsAdaptive {
		for _, p := range g.Pools {
			if m, ok := gaugeMetrics[poolHealthStatusMetricName]; ok {
				m.With(
					prometheus.Labels{
						"zone":               name,
						"account":            account,
						"load_balancer_name": g.LbName,
						"pool_name":          p.PoolName,
					}).Set(float64(p.Healthy))
			}
		}
	}
}
