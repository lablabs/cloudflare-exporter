package main

import (
	"os"
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

	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now().Add(time.Duration(-60) * time.Second)
	then := now.Add(time.Duration(-60) * time.Second)
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
