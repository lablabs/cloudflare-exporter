package main

import (
	"os"
	"strconv"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
)

func fetchZones() []cloudflare.Zone {

	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	z, err := api.ListZones()
	if err != nil {
		log.Fatal(err)
	}

	return z

}

func fetchZoneTotals(zoneID string) *cloudflare.ZoneAnalytics {

	var timeWindow int64
	if os.Getenv("TIME_WINDOW_SECONDS") == "" {
		timeWindow = 60
	} else {
		timeWindow, _ = strconv.ParseInt(os.Getenv("TIME_WINDOW_SECONDS"), 10, 64)
	}

	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	then := now.Add(time.Duration(-timeWindow) * time.Second)
	continuous := false

	options := &cloudflare.ZoneAnalyticsOptions{
		Since:      &then,
		Until:      &now,
		Continuous: &continuous,
	}

	zone, err := api.ZoneAnalyticsDashboard(zoneID, *options)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	zoneTotals := zone.Totals

	return &zoneTotals
}
