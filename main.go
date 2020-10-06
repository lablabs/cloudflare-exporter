package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/biter777/countries"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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
)

func fetchMetrics() {
	var wg sync.WaitGroup
	zones := fetchZones()
	for _, z := range zones {
		wg.Add(1)
		go func(ID string, name string) {
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

		}(z.ID, z.Name)
	}
	wg.Wait()
}

func main() {
	go func() {
		for ; true; <-time.NewTicker(60 * time.Second).C {
			go fetchMetrics()
		}
	}() //This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
