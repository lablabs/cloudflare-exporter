package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func fetchMetrics() {
	var wg sync.WaitGroup
	zones := fetchZones()
	for _, z := range zones {
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
