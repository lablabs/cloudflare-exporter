package main

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/biter777/countries"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricKind string

func (mn MetricKind) String() string {
	return string(mn)
}

const (
	zoneRequestTotalMetricName                   MetricKind = "cloudflare_zone_requests_total"
	zoneRequestCachedMetricName                  MetricKind = "cloudflare_zone_requests_cached"
	zoneRequestSSLEncryptedMetricName            MetricKind = "cloudflare_zone_requests_ssl_encrypted"
	zoneRequestContentTypeMetricName             MetricKind = "cloudflare_zone_requests_content_type"
	zoneRequestCountryMetricName                 MetricKind = "cloudflare_zone_requests_country"
	zoneRequestHTTPStatusMetricName              MetricKind = "cloudflare_zone_requests_status"
	zoneRequestBrowserMapMetricName              MetricKind = "cloudflare_zone_requests_browser_map_page_views_count"
	zoneRequestOriginStatusCountryHostMetricName MetricKind = "cloudflare_zone_requests_origin_status_country_host"
	zoneRequestStatusCountryHostMetricName       MetricKind = "cloudflare_zone_requests_status_country_host"
	zoneBandwidthTotalMetricName                 MetricKind = "cloudflare_zone_bandwidth_total"
	zoneBandwidthCachedMetricName                MetricKind = "cloudflare_zone_bandwidth_cached"
	zoneBandwidthSSLEncryptedMetricName          MetricKind = "cloudflare_zone_bandwidth_ssl_encrypted"
	zoneBandwidthContentTypeMetricName           MetricKind = "cloudflare_zone_bandwidth_content_type"
	zoneBandwidthCountryMetricName               MetricKind = "cloudflare_zone_bandwidth_country"
	zoneThreatsTotalMetricName                   MetricKind = "cloudflare_zone_threats_total"
	zoneThreatsCountryMetricName                 MetricKind = "cloudflare_zone_threats_country"
	zoneThreatsTypeMetricName                    MetricKind = "cloudflare_zone_threats_type"
	zonePageviewsTotalMetricName                 MetricKind = "cloudflare_zone_pageviews_total"
	zoneUniquesTotalMetricName                   MetricKind = "cloudflare_zone_uniques_total"
	zoneColocationVisitsMetricName               MetricKind = "cloudflare_zone_colocation_visits"
	zoneColocationEdgeResponseBytesMetricName    MetricKind = "cloudflare_zone_colocation_edge_response_bytes"
	zoneColocationRequestsTotalMetricName        MetricKind = "cloudflare_zone_colocation_requests_total"
	zoneFirewallEventsCountMetricName            MetricKind = "cloudflare_zone_firewall_events_count"
	zoneHealthCheckEventsOriginCountMetricName   MetricKind = "cloudflare_zone_health_check_events_origin_count"
	workerRequestsMetricName                     MetricKind = "cloudflare_worker_requests_count"
	workerErrorsMetricName                       MetricKind = "cloudflare_worker_errors_count"
	workerCPUTimeMetricName                      MetricKind = "cloudflare_worker_cpu_time"
	workerDurationMetricName                     MetricKind = "cloudflare_worker_duration"
	poolHealthStatusMetricName                   MetricKind = "cloudflare_zone_pool_health_status"
	poolRequestsTotalMetricName                  MetricKind = "cloudflare_zone_pool_requests_total"
)

type MetricsSet map[MetricKind]struct{}

func (ms MetricsSet) Has(mn MetricKind) bool {
	_, exists := ms[mn]
	return exists
}

func (ms MetricsSet) Add(mn MetricKind) {
	ms[mn] = struct{}{}
}

var (
	// Requests
	zoneRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestTotalMetricName.String(),
		Help: "Number of requests for zone",
	}, []string{"zone"},
	)

	zoneRequestCached = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestCachedMetricName.String(),
		Help: "Number of cached requests for zone",
	}, []string{"zone"},
	)

	zoneRequestSSLEncrypted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestSSLEncryptedMetricName.String(),
		Help: "Number of encrypted requests for zone",
	}, []string{"zone"},
	)

	zoneRequestContentType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestContentTypeMetricName.String(),
		Help: "Number of request for zone per content type",
	}, []string{"zone", "content_type"},
	)

	zoneRequestCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestCountryMetricName.String(),
		Help: "Number of request for zone per country",
	}, []string{"zone", "country", "region"},
	)

	zoneRequestHTTPStatus = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestHTTPStatusMetricName.String(),
		Help: "Number of request for zone per HTTP status",
	}, []string{"zone", "status"},
	)

	zoneRequestBrowserMap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestBrowserMapMetricName.String(),
		Help: "Number of successful requests for HTML pages per zone",
	}, []string{"zone", "family"},
	)

	zoneRequestOriginStatusCountryHost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestOriginStatusCountryHostMetricName.String(),
		Help: "Count of not cached requests for zone per origin HTTP status per country per host",
	}, []string{"zone", "status", "country", "host"},
	)

	zoneRequestStatusCountryHost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestStatusCountryHostMetricName.String(),
		Help: "Count of requests for zone per edge HTTP status per country per host",
	}, []string{"zone", "status", "country", "host"},
	)

	zoneBandwidthTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthTotalMetricName.String(),
		Help: "Total bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthCached = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthCachedMetricName.String(),
		Help: "Cached bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthSSLEncrypted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthSSLEncryptedMetricName.String(),
		Help: "Encrypted bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthContentType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthContentTypeMetricName.String(),
		Help: "Bandwidth per zone per content type",
	}, []string{"zone", "content_type"},
	)

	zoneBandwidthCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthCountryMetricName.String(),
		Help: "Bandwidth per country per zone",
	}, []string{"zone", "country", "region"},
	)

	zoneThreatsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsTotalMetricName.String(),
		Help: "Threats per zone",
	}, []string{"zone"},
	)

	zoneThreatsCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsCountryMetricName.String(),
		Help: "Threats per zone per country",
	}, []string{"zone", "country", "region"},
	)

	zoneThreatsType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsTypeMetricName.String(),
		Help: "Threats per zone per type",
	}, []string{"zone", "type"},
	)

	zonePageviewsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zonePageviewsTotalMetricName.String(),
		Help: "Pageviews per zone",
	}, []string{"zone"},
	)

	zoneUniquesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneUniquesTotalMetricName.String(),
		Help: "Uniques per zone",
	}, []string{"zone"},
	)

	zoneColocationVisits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationVisitsMetricName.String(),
		Help: "Total visits per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneColocationEdgeResponseBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationEdgeResponseBytesMetricName.String(),
		Help: "Edge response bytes per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneColocationRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationRequestsTotalMetricName.String(),
		Help: "Total requests per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneFirewallEventsCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneFirewallEventsCountMetricName.String(),
		Help: "Count of Firewall events",
	}, []string{"zone", "action", "source", "host", "country"},
	)

	zoneHealthCheckEventsOriginCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneHealthCheckEventsOriginCountMetricName.String(),
		Help: "Number of Heath check events per region per origin",
	}, []string{"zone", "health_status", "origin_ip", "region", "fqdn"},
	)

	workerRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: workerRequestsMetricName.String(),
		Help: "Number of requests sent to worker by script name",
	}, []string{"script_name"},
	)

	workerErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: workerErrorsMetricName.String(),
		Help: "Number of errors by script name",
	}, []string{"script_name"},
	)

	workerCPUTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: workerCPUTimeMetricName.String(),
		Help: "CPU time quantiles by script name",
	}, []string{"script_name", "quantile"},
	)

	workerDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: workerDurationMetricName.String(),
		Help: "Duration quantiles by script name (GB*s)",
	}, []string{"script_name", "quantile"},
	)

	poolHealthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: poolHealthStatusMetricName.String(),
		Help: "Reports the health of a pool, 1 for healthy, 0 for unhealthy.",
	},
		[]string{"zone", "load_balancer_name", "pool_name"},
	)

	poolRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: poolRequestsTotalMetricName.String(),
		Help: "Requests per pool",
	},
		[]string{"zone", "load_balancer_name", "pool_name", "origin_name"},
	)
)

func buildAllMetricsSet() MetricsSet {
	allMetricsSet := MetricsSet{}
	allMetricsSet.Add(zoneRequestTotalMetricName)
	allMetricsSet.Add(zoneRequestCachedMetricName)
	allMetricsSet.Add(zoneRequestSSLEncryptedMetricName)
	allMetricsSet.Add(zoneRequestContentTypeMetricName)
	allMetricsSet.Add(zoneRequestCountryMetricName)
	allMetricsSet.Add(zoneRequestHTTPStatusMetricName)
	allMetricsSet.Add(zoneRequestBrowserMapMetricName)
	allMetricsSet.Add(zoneRequestOriginStatusCountryHostMetricName)
	allMetricsSet.Add(zoneRequestStatusCountryHostMetricName)
	allMetricsSet.Add(zoneBandwidthTotalMetricName)
	allMetricsSet.Add(zoneBandwidthCachedMetricName)
	allMetricsSet.Add(zoneBandwidthSSLEncryptedMetricName)
	allMetricsSet.Add(zoneBandwidthContentTypeMetricName)
	allMetricsSet.Add(zoneBandwidthCountryMetricName)
	allMetricsSet.Add(zoneThreatsTotalMetricName)
	allMetricsSet.Add(zoneThreatsCountryMetricName)
	allMetricsSet.Add(zoneThreatsTypeMetricName)
	allMetricsSet.Add(zonePageviewsTotalMetricName)
	allMetricsSet.Add(zoneUniquesTotalMetricName)
	allMetricsSet.Add(zoneColocationVisitsMetricName)
	allMetricsSet.Add(zoneColocationEdgeResponseBytesMetricName)
	allMetricsSet.Add(zoneColocationRequestsTotalMetricName)
	allMetricsSet.Add(zoneFirewallEventsCountMetricName)
	allMetricsSet.Add(zoneHealthCheckEventsOriginCountMetricName)
	allMetricsSet.Add(workerRequestsMetricName)
	allMetricsSet.Add(workerErrorsMetricName)
	allMetricsSet.Add(workerCPUTimeMetricName)
	allMetricsSet.Add(workerDurationMetricName)
	allMetricsSet.Add(poolHealthStatusMetricName)
	allMetricsSet.Add(poolRequestsTotalMetricName)
	return allMetricsSet
}

func buildDeniedMetricsSet(metricsDenylist []string) (MetricsSet, error) {
	deniedMetricsSet := MetricsSet{}
	allMetricsSet := buildAllMetricsSet()
	for _, metric := range metricsDenylist {
		if !allMetricsSet.Has(MetricKind(metric)) {
			return nil, fmt.Errorf("metric %s doesn't exists", metric)
		}
		deniedMetricsSet.Add(MetricKind(metric))
	}
	return deniedMetricsSet, nil
}

func mustRegisterMetrics(deniedMetrics MetricsSet) {
	if !deniedMetrics.Has(zoneRequestTotalMetricName) {
		prometheus.MustRegister(zoneRequestTotal)
	}
	if !deniedMetrics.Has(zoneRequestCachedMetricName) {
		prometheus.MustRegister(zoneRequestCached)
	}
	if !deniedMetrics.Has(zoneRequestSSLEncryptedMetricName) {
		prometheus.MustRegister(zoneRequestSSLEncrypted)
	}
	if !deniedMetrics.Has(zoneRequestContentTypeMetricName) {
		prometheus.MustRegister(zoneRequestContentType)
	}
	if !deniedMetrics.Has(zoneRequestCountryMetricName) {
		prometheus.MustRegister(zoneRequestCountry)
	}
	if !deniedMetrics.Has(zoneRequestHTTPStatusMetricName) {
		prometheus.MustRegister(zoneRequestHTTPStatus)
	}
	if !deniedMetrics.Has(zoneRequestBrowserMapMetricName) {
		prometheus.MustRegister(zoneRequestBrowserMap)
	}
	if !deniedMetrics.Has(zoneRequestOriginStatusCountryHostMetricName) {
		prometheus.MustRegister(zoneRequestOriginStatusCountryHost)
	}
	if !deniedMetrics.Has(zoneRequestStatusCountryHostMetricName) {
		prometheus.MustRegister(zoneRequestStatusCountryHost)
	}
	if !deniedMetrics.Has(zoneBandwidthTotalMetricName) {
		prometheus.MustRegister(zoneBandwidthTotal)
	}
	if !deniedMetrics.Has(zoneBandwidthCachedMetricName) {
		prometheus.MustRegister(zoneBandwidthCached)
	}
	if !deniedMetrics.Has(zoneBandwidthSSLEncryptedMetricName) {
		prometheus.MustRegister(zoneBandwidthSSLEncrypted)
	}
	if !deniedMetrics.Has(zoneBandwidthContentTypeMetricName) {
		prometheus.MustRegister(zoneBandwidthContentType)
	}
	if !deniedMetrics.Has(zoneBandwidthCountryMetricName) {
		prometheus.MustRegister(zoneBandwidthCountry)
	}
	if !deniedMetrics.Has(zoneThreatsTotalMetricName) {
		prometheus.MustRegister(zoneThreatsTotal)
	}
	if !deniedMetrics.Has(zoneThreatsCountryMetricName) {
		prometheus.MustRegister(zoneThreatsCountry)
	}
	if !deniedMetrics.Has(zoneThreatsTypeMetricName) {
		prometheus.MustRegister(zoneThreatsType)
	}
	if !deniedMetrics.Has(zonePageviewsTotalMetricName) {
		prometheus.MustRegister(zonePageviewsTotal)
	}
	if !deniedMetrics.Has(zoneUniquesTotalMetricName) {
		prometheus.MustRegister(zoneUniquesTotal)
	}
	if !deniedMetrics.Has(zoneColocationVisitsMetricName) {
		prometheus.MustRegister(zoneColocationVisits)
	}
	if !deniedMetrics.Has(zoneColocationEdgeResponseBytesMetricName) {
		prometheus.MustRegister(zoneColocationEdgeResponseBytes)
	}
	if !deniedMetrics.Has(zoneColocationRequestsTotalMetricName) {
		prometheus.MustRegister(zoneColocationRequestsTotal)
	}
	if !deniedMetrics.Has(zoneFirewallEventsCountMetricName) {
		prometheus.MustRegister(zoneFirewallEventsCount)
	}
	if !deniedMetrics.Has(zoneHealthCheckEventsOriginCountMetricName) {
		prometheus.MustRegister(zoneHealthCheckEventsOriginCount)
	}
	if !deniedMetrics.Has(workerRequestsMetricName) {
		prometheus.MustRegister(workerRequests)
	}
	if !deniedMetrics.Has(workerErrorsMetricName) {
		prometheus.MustRegister(workerErrors)
	}
	if !deniedMetrics.Has(workerCPUTimeMetricName) {
		prometheus.MustRegister(workerCPUTime)
	}
	if !deniedMetrics.Has(workerDurationMetricName) {
		prometheus.MustRegister(workerDuration)
	}
	if !deniedMetrics.Has(poolHealthStatusMetricName) {
		prometheus.MustRegister(poolHealthStatus)
	}
	if !deniedMetrics.Has(poolRequestsTotalMetricName) {
		prometheus.MustRegister(poolRequestsTotal)
	}
}

func fetchWorkerAnalytics(account cloudflare.Account, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	r, err := fetchWorkerTotals(account.ID)
	if err != nil {
		return
	}

	for _, a := range r.Viewer.Accounts {
		for _, w := range a.WorkersInvocationsAdaptive {
			workerRequests.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName}).Add(float64(w.Sum.Requests))
			workerErrors.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName}).Add(float64(w.Sum.Errors))
			workerCPUTime.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P50"}).Set(float64(w.Quantiles.CPUTimeP50))
			workerCPUTime.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P75"}).Set(float64(w.Quantiles.CPUTimeP75))
			workerCPUTime.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P99"}).Set(float64(w.Quantiles.CPUTimeP99))
			workerCPUTime.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P999"}).Set(float64(w.Quantiles.CPUTimeP999))
			workerDuration.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P50"}).Set(float64(w.Quantiles.DurationP50))
			workerDuration.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P75"}).Set(float64(w.Quantiles.DurationP75))
			workerDuration.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P99"}).Set(float64(w.Quantiles.DurationP99))
			workerDuration.With(prometheus.Labels{"script_name": w.Dimensions.ScriptName, "quantile": "P999"}).Set(float64(w.Quantiles.DurationP999))
		}
	}
}

func fetchZoneColocationAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// Colocation metrics are not available in non-enterprise zones
	if cfgFreeTier {
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
		name := findZoneName(zones, z.ZoneTag)
		for _, c := range cg {
			zoneColocationVisits.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.Visits))
			zoneColocationEdgeResponseBytes.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.EdgeResponseBytes))
			zoneColocationRequestsTotal.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Count))
		}
	}
}

func fetchZoneAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// None of the below referenced metrics are available in the free tier
	if cfgFreeTier {
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
		name := findZoneName(zones, z.ZoneTag)
		addHTTPGroups(&z, name)
		addFirewallGroups(&z, name)
		addHealthCheckGroups(&z, name)
		addHTTPAdaptiveGroups(&z, name)
	}
}

func addHTTPGroups(z *zoneResp, name string) {
	// Nothing to do.
	if len(z.HTTP1mGroups) == 0 {
		return
	}
	zt := z.HTTP1mGroups[0]

	zoneRequestTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.Requests))
	zoneRequestCached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.CachedRequests))
	zoneRequestSSLEncrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.EncryptedRequests))

	for _, ct := range zt.Sum.ContentType {
		zoneRequestContentType.With(prometheus.Labels{"zone": name, "content_type": ct.EdgeResponseContentType}).Add(float64(ct.Requests))
		zoneBandwidthContentType.With(prometheus.Labels{"zone": name, "content_type": ct.EdgeResponseContentType}).Add(float64(ct.Bytes))
	}

	for _, country := range zt.Sum.Country {
		c := countries.ByName(country.ClientCountryName)
		region := c.Info().Region.Info().Name

		zoneRequestCountry.With(prometheus.Labels{"zone": name, "country": country.ClientCountryName, "region": region}).Add(float64(country.Requests))
		zoneBandwidthCountry.With(prometheus.Labels{"zone": name, "country": country.ClientCountryName, "region": region}).Add(float64(country.Bytes))
		zoneThreatsCountry.With(prometheus.Labels{"zone": name, "country": country.ClientCountryName, "region": region}).Add(float64(country.Threats))
	}

	for _, status := range zt.Sum.ResponseStatus {
		zoneRequestHTTPStatus.With(prometheus.Labels{"zone": name, "status": strconv.Itoa(status.EdgeResponseStatus)}).Add(float64(status.Requests))
	}

	for _, browser := range zt.Sum.BrowserMap {
		zoneRequestBrowserMap.With(prometheus.Labels{"zone": name, "family": browser.UaBrowserFamily}).Add(float64(browser.PageViews))
	}

	zoneBandwidthTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.Bytes))
	zoneBandwidthCached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.CachedBytes))
	zoneBandwidthSSLEncrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.EncryptedBytes))

	zoneThreatsTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.Threats))

	for _, t := range zt.Sum.ThreatPathing {
		zoneThreatsType.With(prometheus.Labels{"zone": name, "type": t.Name}).Add(float64(t.Requests))
	}

	zonePageviewsTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Sum.PageViews))

	// Uniques
	zoneUniquesTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Unique.Uniques))
}

func addFirewallGroups(z *zoneResp, name string) {
	// Nothing to do.
	if len(z.FirewallEventsAdaptiveGroups) == 0 {
		return
	}

	for _, g := range z.FirewallEventsAdaptiveGroups {
		zoneFirewallEventsCount.With(
			prometheus.Labels{
				"zone":    name,
				"action":  g.Dimensions.Action,
				"source":  g.Dimensions.Source,
				"host":    g.Dimensions.ClientRequestHTTPHost,
				"country": g.Dimensions.ClientCountryName,
			}).Add(float64(g.Count))
	}
}

func addHealthCheckGroups(z *zoneResp, name string) {
	if len(z.HealthCheckEventsAdaptiveGroups) == 0 {
		return
	}

	for _, g := range z.HealthCheckEventsAdaptiveGroups {
		zoneHealthCheckEventsOriginCount.With(
			prometheus.Labels{
				"zone":          name,
				"health_status": g.Dimensions.HealthStatus,
				"origin_ip":     g.Dimensions.OriginIP,
				"region":        g.Dimensions.Region,
				"fqdn":          g.Dimensions.Fqdn,
			}).Add(float64(g.Count))
	}
}

func addHTTPAdaptiveGroups(z *zoneResp, name string) {

	for _, g := range z.HTTPRequestsAdaptiveGroups {
		zoneRequestOriginStatusCountryHost.With(
			prometheus.Labels{
				"zone":    name,
				"status":  strconv.Itoa(int(g.Dimensions.OriginResponseStatus)),
				"country": g.Dimensions.ClientCountryName,
				"host":    g.Dimensions.ClientRequestHTTPHost,
			}).Add(float64(g.Count))
	}

	for _, g := range z.HTTPRequestsEdgeCountryHost {
		zoneRequestStatusCountryHost.With(
			prometheus.Labels{
				"zone":    name,
				"status":  strconv.Itoa(int(g.Dimensions.EdgeResponseStatus)),
				"country": g.Dimensions.ClientCountryName,
				"host":    g.Dimensions.ClientRequestHTTPHost,
			}).Add(float64(g.Count))
	}

}

func fetchLoadBalancerAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// None of the below referenced metrics are available in the free tier
	if cfgFreeTier {
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
		name := findZoneName(zones, lb.ZoneTag)
		addLoadBalancingRequestsAdaptive(&lb, name)
		addLoadBalancingRequestsAdaptiveGroups(&lb, name)
	}
}

func addLoadBalancingRequestsAdaptiveGroups(z *lbResp, name string) {

	for _, g := range z.LoadBalancingRequestsAdaptiveGroups {
		poolRequestsTotal.With(
			prometheus.Labels{
				"zone":               name,
				"load_balancer_name": g.Dimensions.LbName,
				"pool_name":          g.Dimensions.SelectedPoolName,
				"origin_name":        g.Dimensions.SelectedOriginName,
			}).Add(float64(g.Count))
	}
}

func addLoadBalancingRequestsAdaptive(z *lbResp, name string) {

	for _, g := range z.LoadBalancingRequestsAdaptive {
		for _, p := range g.Pools {
			poolHealthStatus.With(
				prometheus.Labels{
					"zone":               name,
					"load_balancer_name": g.LbName,
					"pool_name":          p.PoolName,
				}).Set(float64(p.Healthy))
		}
	}

}
