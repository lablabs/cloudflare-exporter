package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/namsral/flag"
	"github.com/nelkinda/health-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	cfgListen          = ":8080"
	cfgCfAPIKey        = ""
	cfgCfAPIEmail      = ""
	cfgCfAPIToken      = ""
	cfgMetricsPath     = "/metrics"
	cfgZones           = ""
	cfgExcludeZones    = ""
	cfgScrapeDelay     = 300
	cfgFreeTier        = false
	cfgBatchSize       = 10
	cfgMetricsDenylist = ""
)

func getTargetZones() []string {
	var zoneIDs []string

	if len(cfgZones) > 0 {
		zoneIDs = strings.Split(cfgZones, ",")
	} else {
		// deprecated
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "ZONE_") {
				split := strings.SplitN(e, "=", 2)
				zoneIDs = append(zoneIDs, split[1])
			}
		}
	}
	return zoneIDs
}

func getExcludedZones() []string {
	var zoneIDs []string

	if len(cfgExcludeZones) > 0 {
		zoneIDs = strings.Split(cfgExcludeZones, ",")
	}
	return zoneIDs
}

func filterZones(all []cloudflare.Zone, target []string) []cloudflare.Zone {
	var filtered []cloudflare.Zone

	if (len(target)) == 0 {
		return all
	}

	for _, tz := range target {
		for _, z := range all {
			if tz == z.ID {
				filtered = append(filtered, z)
				log.Info("Filtering zone: ", z.ID, " ", z.Name)
			}
		}
	}

	return filtered
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func filterExcludedZones(all []cloudflare.Zone, exclude []string) []cloudflare.Zone {
	var filtered []cloudflare.Zone

	if (len(exclude)) == 0 {
		return all
	}

	for _, z := range all {
		if contains(exclude, z.ID) {
			log.Info("Exclude zone: ", z.ID, " ", z.Name)
		} else {
			filtered = append(filtered, z)
		}
	}

	return filtered
}

func fetchMetrics(deniedMetricsSet MetricsSet) {
	var wg sync.WaitGroup
	zones := fetchZones()
	accounts := fetchAccounts()
	filteredZones := filterExcludedZones(filterZones(zones, getTargetZones()), getExcludedZones())

	for _, a := range accounts {
		go fetchWorkerAnalytics(a, &wg)
	}

	// Make requests in groups of cfgBatchSize to avoid rate limit
	// 10 is the maximum amount of zones you can request at once
	for len(filteredZones) > 0 {
		sliceLength := cfgBatchSize
		if len(filteredZones) < cfgBatchSize {
			sliceLength = len(filteredZones)
		}

		targetZones := filteredZones[:sliceLength]
		filteredZones = filteredZones[len(targetZones):]

		go fetchZoneAnalytics(targetZones, &wg, deniedMetricsSet)
		go fetchZoneColocationAnalytics(targetZones, &wg, deniedMetricsSet)
		go fetchLoadBalancerAnalytics(targetZones, &wg, deniedMetricsSet)
	}

	wg.Wait()
}

func main() {
	flag.StringVar(&cfgListen, "listen", cfgListen, "listen on addr:port ( default :8080), omit addr to listen on all interfaces")
	flag.StringVar(&cfgMetricsPath, "metrics_path", cfgMetricsPath, "path for metrics, default /metrics")
	flag.StringVar(&cfgCfAPIKey, "cf_api_key", cfgCfAPIKey, "cloudflare api key, works with api_email flag")
	flag.StringVar(&cfgCfAPIEmail, "cf_api_email", cfgCfAPIEmail, "cloudflare api email, works with api_key flag")
	flag.StringVar(&cfgCfAPIToken, "cf_api_token", cfgCfAPIToken, "cloudflare api token (preferred)")
	flag.StringVar(&cfgZones, "cf_zones", cfgZones, "cloudflare zones to export, comma delimited list")
	flag.StringVar(&cfgExcludeZones, "cf_exclude_zones", cfgExcludeZones, "cloudflare zones to exclude, comma delimited list")
	flag.IntVar(&cfgScrapeDelay, "scrape_delay", cfgScrapeDelay, "scrape delay in seconds, defaults to 300")
	flag.IntVar(&cfgBatchSize, "cf_batch_size", cfgBatchSize, "cloudflare zones batch size (1-10), defaults to 10")
	flag.BoolVar(&cfgFreeTier, "free_tier", cfgFreeTier, "scrape only metrics included in free plan")
	flag.StringVar(&cfgMetricsDenylist, "metrics_denylist", cfgMetricsDenylist, "metrics to not expose, comma delimited list")
	flag.Parse()
	if !(len(cfgCfAPIToken) > 0 || (len(cfgCfAPIEmail) > 0 && len(cfgCfAPIKey) > 0)) {
		log.Fatal("Please provide CF_API_KEY+CF_API_EMAIL or CF_API_TOKEN")
	}
	if cfgBatchSize < 1 || cfgBatchSize > 10 {
		log.Fatal("CF_BATCH_SIZE must be between 1 and 10")
	}
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	metricsDenylist := []string{}
	if len(cfgMetricsDenylist) > 0 {
		metricsDenylist = strings.Split(cfgMetricsDenylist, ",")
	}
	deniedMetricsSet, err := buildDeniedMetricsSet(metricsDenylist)
	if err != nil {
		log.Fatal(err)
	}
	mustRegisterMetrics(deniedMetricsSet)

	go func() {
		for ; true; <-time.NewTicker(60 * time.Second).C {
			go fetchMetrics(deniedMetricsSet)
		}
	}()

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	if !strings.HasPrefix(cfgMetricsPath, "/") {
		cfgMetricsPath = "/" + cfgMetricsPath
	}
	http.Handle(cfgMetricsPath, promhttp.Handler())
	h := health.New(health.Health{})
	http.HandleFunc("/health", h.Handler)
	log.Info("Beginning to serve on port", cfgListen, ", metrics path ", cfgMetricsPath)
	log.Fatal(http.ListenAndServe(cfgListen, nil))
}
