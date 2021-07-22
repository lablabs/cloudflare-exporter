package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	cfgListen      = ":8080"
	cfgCfAPIKey    = ""
	cfgCfAPIEmail  = ""
	cfgCfAPIToken  = ""
	cfgMetricsPath = "/metrics"
	cfgZones       = ""
	cfgScrapeDelay = 300
)

func getTargetZones() []string {
	var zoneIDs []string

	if len(cfgZones) > 0 {
		zoneIDs = strings.Split(cfgZones, ",")
	} else {
		//depricated
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "ZONE_") {
				split := strings.SplitN(e, "=", 2)
				zoneIDs = append(zoneIDs, split[1])
			}
		}
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

func fetchMetrics() {
	var wg sync.WaitGroup
	zones := fetchZones()

	filteredZones := filterZones(zones, getTargetZones())

	// Make requests in groups of 10 to avoid rate limit
	// 10 is the maximum amount of zones you can request at once
	for len(filteredZones) > 0 {
		sliceLength := 10
		if len(filteredZones) < 10 {
			sliceLength = len(filteredZones)
		}

		targetZones := filteredZones[:sliceLength]
		filteredZones = filteredZones[len(targetZones):]

		go fetchZoneAnalytics(targetZones, &wg)
		go fetchZoneColocationAnalytics(targetZones, &wg)
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
	flag.IntVar(&cfgScrapeDelay, "scrape_delay", cfgScrapeDelay , "scrape delay in seconds, defaults to 180")
	flag.Parse()
	if !(len(cfgCfAPIToken) > 0 || (len(cfgCfAPIEmail) > 0 && len(cfgCfAPIKey) > 0)) {
		log.Fatal("Please provide CF_API_KEY+CF_API_EMAIL or CF_API_TOKEN")
	}
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	go func() {
		for ; true; <-time.NewTicker(60 * time.Second).C {
			go fetchMetrics()
		}
	}()

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	if !strings.HasPrefix(cfgMetricsPath, "/") {
		cfgMetricsPath = "/" + cfgMetricsPath
	}
	http.Handle(cfgMetricsPath, promhttp.Handler())
	log.Info("Beginning to serve on port", cfgListen, ", metrics path ", cfgMetricsPath)
	log.Fatal(http.ListenAndServe(cfgListen, nil))
}
