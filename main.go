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
	cfg_listen       = ":8080"
	cfg_cf_api_key   = ""
	cfg_cf_api_email = ""
	cfg_cf_api_token = ""
	cfg_metrics_path = "/metrics"
)

func getTargetZones() []string {
	var zoneIDs []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ZONE_") {
			split := strings.SplitN(e, "=", 2)
			zoneIDs = append(zoneIDs, split[1])
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
	flag.StringVar(&cfg_listen, "listen", cfg_listen, "listen on addr:port ( default :8080), omit addr to listen on all interfaces")
	flag.StringVar(&cfg_metrics_path, "metrics_path", cfg_metrics_path, "path for metrics, default /metrics")
	flag.StringVar(&cfg_cf_api_key, "cf_api_key", cfg_cf_api_key, "cloudflare api key, works with api_email flag")
	flag.StringVar(&cfg_cf_api_email, "cf_api_email", cfg_cf_api_email, "cloudflare api email, works with api_key flag")
	flag.StringVar(&cfg_cf_api_token, "cf_api_token", cfg_cf_api_token, "cloudflare api token (preferred)")
	flag.Parse()
	if !(len(cfg_cf_api_token) > 0 || (len(cfg_cf_api_email) > 0 && len(cfg_cf_api_key) > 0)) {
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
	if !strings.HasPrefix(cfg_metrics_path, "/") {
		cfg_metrics_path = "/" + cfg_metrics_path
	}
	http.Handle(cfg_metrics_path, promhttp.Handler())
	log.Info("Beginning to serve on ", cfg_listen, ", metrics path ", cfg_metrics_path)
	log.Fatal(http.ListenAndServe(cfg_listen, nil))
}
