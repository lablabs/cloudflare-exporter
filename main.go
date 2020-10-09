package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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

	for _, z := range filterZones(zones, getTargetZones()) {
		go fetchZoneAnalytics(z.ID, z.Name, &wg)
		go fetchZoneColocationAnalytics(z.ID, z.Name, &wg)
	}
	wg.Wait()
}

func main() {
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
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
