package main

import (
	"strconv"
	"sync"

	"github.com/biter777/countries"
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

	zoneColocationRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_requests_total",
		Help: "Total requests per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneColocationRequestsCached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_requests_cached",
		Help: "Total cached requests per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneColocationBandwidthTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_bandwidth_total",
		Help: "Total bandwidth per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneColocationBandwidthCached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_bandwidth_cached",
		Help: "Total cached bandwidth per colocation",
	}, []string{"zone", "colocation"},
	)

	zoneColocationResponseStatus = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_response_status",
		Help: "HTTP response status per colocation",
	}, []string{"zone", "colocation", "status"},
	)

	zoneColocationRequestsByCountry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_requests_country",
		Help: "Requests per colocation per country",
	}, []string{"zone", "colocation", "country", "region"},
	)

	zoneColocationThreatsByCountry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_threats_country",
		Help: "Threats per colocation per country",
	}, []string{"zone", "colocation", "country", "region"},
	)
)

func fetchZoneColocationAnalytics(ID string, name string, wg *sync.WaitGroup) {
	wg.Add(1)
	cg := fetchColoTotals(ID).Viewer.Zones[0].ColoGroups

	for _, c := range cg {
		zoneColocationRequestsTotal.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.Requests))
		zoneColocationRequestsCached.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.CachedRequests))
		zoneColocationBandwidthTotal.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.Bytes))
		zoneColocationBandwidthCached.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode}).Add(float64(c.Sum.CachedBytes))

		for _, s := range c.Sum.ResponseStatusMap {
			zoneColocationResponseStatus.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "status": strconv.Itoa(s.EdgeResponseStatus)}).Add(float64(s.Requests))
		}

		for _, country := range c.Sum.CountryMap {
			region := countries.ByName(country.ClientCountryName).Info().Region.Info().Name

			zoneColocationRequestsByCountry.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "country": country.ClientCountryName, "region": region}).Add(float64(country.Requests))
			zoneColocationRequestsByCountry.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "country": country.ClientCountryName, "region": region}).Add(float64(country.Threats))
		}
	}

	defer wg.Done()
}

func fetchZoneAnalytics(ID string, name string, wg *sync.WaitGroup) {
	wg.Add(1)
	zt := fetchZoneTotals(ID)

	// Requests
	zoneRequestTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Requests.All))
	zoneRequestCached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Requests.Cached))
	zoneRequestUncached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Requests.Uncached))
	zoneRequestSSLEncrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Requests.SSL.Encrypted))
	zoneRequestSSLUnencrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Requests.SSL.Unencrypted))

	for ct, value := range zt.Requests.ContentType {
		zoneRequestContentType.With(prometheus.Labels{"zone": name, "content_type": ct}).Add(float64(value))
	}

	for country, value := range zt.Requests.Country {
		c := countries.ByName(country)
		region := c.Info().Region.Info().Name
		zoneRequestCountry.With(prometheus.Labels{"zone": name, "country": country, "region": region}).Add(float64(value))
	}

	for status, value := range zt.Requests.HTTPStatus {
		zoneRequestHTTPStatus.With(prometheus.Labels{"zone": name, "status": status}).Add(float64(value))
	}

	// Bandwidth

	zoneBandwidthTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Bandwidth.All))
	zoneBandwidthCached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Bandwidth.Cached))
	zoneBandwidthUncached.With(prometheus.Labels{"zone": name}).Add(float64(zt.Bandwidth.Uncached))
	zoneBandwidthSSLEncrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Bandwidth.SSL.Encrypted))
	zoneBandwidthSSLUnencrypted.With(prometheus.Labels{"zone": name}).Add(float64(zt.Bandwidth.SSL.Unencrypted))

	for ct, value := range zt.Bandwidth.ContentType {
		zoneBandwidthContentType.With(prometheus.Labels{"zone": name, "content_type": ct}).Add(float64(value))
	}

	for country, value := range zt.Bandwidth.Country {
		c := countries.ByName(country)
		region := c.Info().Region.Info().Name
		zoneBandwidthCountry.With(prometheus.Labels{"zone": name, "country": country, "region": region}).Add(float64(value))
	}

	// Threats
	zoneThreatsTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Threats.All))

	for country, value := range zt.Threats.Country {
		c := countries.ByName(country)
		region := c.Info().Region.Info().Name
		zoneThreatsCountry.With(prometheus.Labels{"zone": name, "country": country, "region": region}).Add(float64(value))
	}

	for t, value := range zt.Threats.Type {
		zoneThreatsType.With(prometheus.Labels{"zone": name, "type": t}).Add(float64(value))
	}

	// Pageviews

	zonePageviewsTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Pageviews.All))

	for se, value := range zt.Pageviews.SearchEngines {
		zoneThreatsType.With(prometheus.Labels{"zone": name, "searchengine": se}).Add(float64(value))
	}

	// Uniques
	zoneUniquesTotal.With(prometheus.Labels{"zone": name}).Add(float64(zt.Uniques.All))

	defer wg.Done()

}
