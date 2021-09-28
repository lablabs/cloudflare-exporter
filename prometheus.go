package main

import (
	"strconv"
	"sync"

	"github.com/biter777/countries"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Requests
	zoneRequestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_total",
		Help: "Number of requests for zone",
	}, []string{"zone"},
	)

	zoneRequestCached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_cached",
		Help: "Number of cached requests for zone",
	}, []string{"zone"},
	)

	zoneRequestUncached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_uncached",
		Help: "Number of uncached requests for zone",
	}, []string{"zone"},
	)

	zoneRequestSSLEncrypted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_ssl_encrypted",
		Help: "Number of encrypted requests for zone",
	}, []string{"zone"},
	)

	zoneRequestSSLUnencrypted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_ssl_unencrypted",
		Help: "Number of encrypted requests for zone",
	}, []string{"zone"},
	)

	zoneRequestContentType = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_content_type",
		Help: "Number of request for zone per content type",
	}, []string{"zone", "content_type"},
	)

	zoneRequestCountry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_country",
		Help: "Number of request for zone per country",
	}, []string{"zone", "country", "region"},
	)

	zoneRequestHTTPStatus = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_status",
		Help: "Number of request for zone per HTTP status",
	}, []string{"zone", "status"},
	)

	zoneRequestOriginStatusCountryHost = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_origin_status_country_host",
		Help: "Count of not cached requests for zone per origin HTTP status per country per host",
	}, []string{"zone", "status", "country", "host"},
	)

	zoneRequestStatusCountryHost = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_requests_status_country_host",
		Help: "Count of requests for zone per edge HTTP status per country per host",
	}, []string{"zone", "status", "country", "host"},
	)

	zoneBandwidthTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_total",
		Help: "Total bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthCached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_cached",
		Help: "Cached bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthUncached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_uncached",
		Help: "Uncached bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthSSLEncrypted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_ssl_encrypted",
		Help: "Encrypted bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthSSLUnencrypted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_ssl_unencrypted",
		Help: "Unencrypted bandwidth per zone in bytes",
	}, []string{"zone"},
	)

	zoneBandwidthContentType = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_content_type",
		Help: "Bandwidth per zone per content type",
	}, []string{"zone", "content_type"},
	)

	zoneBandwidthCountry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_bandwidth_country",
		Help: "Bandwidth per country per zone",
	}, []string{"zone", "country", "region"},
	)

	zoneThreatsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_threats_total",
		Help: "Threats per zone",
	}, []string{"zone"},
	)

	zoneThreatsCountry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_threats_country",
		Help: "Threats per zone per country",
	}, []string{"zone", "country", "region"},
	)

	zoneThreatsType = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_threats_type",
		Help: "Threats per zone per type",
	}, []string{"zone", "type"},
	)

	zonePageviewsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_pageviews_total",
		Help: "Pageviews per zone",
	}, []string{"zone"},
	)

	zonePageviewsSearchEngines = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_pageviews_search_engines",
		Help: "Pageviews per zone per engine",
	}, []string{"zone", "searchengine"},
	)

	zoneUniquesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_uniques_total",
		Help: "Uniques per zone",
	}, []string{"zone"},
	)

	zoneColocationVisits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_visits",
		Help: "Total visits per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneColocationEdgeResponseBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_edge_response_bytes",
		Help: "Edge response bytes per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneFirewallEventsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_firewall_events_count",
		Help: "Count of Firewall events",
	}, []string{"zone", "action", "source", "host", "country"},
	)

	zoneHealthCheckEventsOriginCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_health_check_events_origin_count",
		Help: "Number of Heath check events per region per origin",
	}, []string{"zone", "health_status", "origin_ip", "region", "fqdn"},
	)

	workerRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_worker_requests_count",
		Help: "Number of requests sent to worker by script name",
	}, []string{"script_name"},
	)

	workerErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_worker_errors_count",
		Help: "Number of errors by script name",
	}, []string{"script_name"},
	)
)

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
		}
	}
}

func fetchZoneColocationAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	zoneIDs := extractZoneIDs(zones)

	r, err := fetchColoTotals(zoneIDs)
	if err != nil {
		return
	}

	for _, z := range r.Viewer.Zones {

		cg := z.ColoGroups
		name := findZoneName(zones, z.ZoneTag)
		for _, c := range cg {
			zoneColocationVisits.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.Visits))
			zoneColocationEdgeResponseBytes.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.EdgeResponseBytes))
		}
	}
}

func fetchZoneAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	zoneIDs := extractZoneIDs(zones)

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
