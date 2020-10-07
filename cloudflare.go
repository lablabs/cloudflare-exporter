package main

import (
	"context"
	"os"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

type cloudflareResponse struct {
	Viewer struct {
		Zones []zoneResp `json:"zones"`
	} `json:"viewer"`
}

type zoneResp struct {
	ColoGroups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
			ColoCode string `json:"coloCode"`
		} `json:"dimensions"`
		Sum struct {
			Bytes          uint64 `json:"bytes"`
			CachedBytes    uint64 `json:"cachedBytes"`
			CachedRequests uint64 `json:"cachedRequests"`
			Requests       uint64 `json:"Requests"`
			CountryMap     []struct {
				ClientCountryName string `json:"clientCountryName"`
				Requests          uint64 `json:"requests"`
				Threats           uint64 `json:"threats"`
			} `json:"countryMap"`
			ResponseStatusMap []struct {
				EdgeResponseStatus int    `json:"edgeResponseStatus"`
				Requests           uint64 `json:"requests"`
			} `json:"responseStatusMap"`
			ThreatPathingMap []struct {
				ThreatPathingName string `json:"threatPathingName"`
				Requests          uint64 `json:"requests"`
			} `json:"threatPathingMap"`
		} `json:"sum"`
	} `json:"httpRequests1mByColoGroups"`

	ZoneTag string `json:"zoneTag"`
}

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

// APIv4 will be deprecated Nov 2020
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

func fetchColoTotals(zoneID string) *cloudflareResponse {

	now := time.Now().Add(time.Duration(-60) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)

	http1mGroupsByColo := graphql.NewRequest(`
	query ($zoneID: String!, $time: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag: $zoneID }) {
				zoneTag

				httpRequests1mByColoGroups(
					limit: $limit
					filter: { datetime: $time }
				) {
					sum {
						requests
						bytes
						countryMap {
							clientCountryName
							requests
							threats
						}
						responseStatusMap {
							edgeResponseStatus
							requests
						}
						cachedRequests
						cachedBytes
						threatPathingMap {
							requests
							threatPathingName
						}
					}
					dimensions {
						coloCode
						datetime
					}
				}
			}
		}
	}
`)

	http1mGroupsByColo.Header.Set("X-AUTH-EMAIL", os.Getenv("CF_API_EMAIL"))
	http1mGroupsByColo.Header.Set("X-AUTH-KEY", os.Getenv("CF_API_KEY"))
	http1mGroupsByColo.Var("limit", 9999)
	http1mGroupsByColo.Var("time", now)
	http1mGroupsByColo.Var("zoneID", zoneID)

	ctx := context.Background()
	graphqlClient := graphql.NewClient("https://api.cloudflare.com/client/v4/graphql/")
	var resp cloudflareResponse
	if err := graphqlClient.Run(ctx, http1mGroupsByColo, &resp); err != nil {
		log.Fatal(err)
	}

	return &resp
}
