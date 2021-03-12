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
	HTTP1mGroups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
		} `json:"dimensions"`
		Unique struct {
			Uniques uint64 `json:"uniques"`
		} `json:"uniq"`
		Sum struct {
			Bytes          uint64 `json:"bytes"`
			CachedBytes    uint64 `json:"cachedBytes"`
			CachedRequests uint64 `json:"cachedRequests"`
			Requests       uint64 `json:"requests"`

			BrowserMap []struct {
				PageViews       uint64 `json:"pageViews"`
				uaBrowserFamily string `json:"uaBrowserFamily"`
			} `json:"browserMap"`
			ClientHTTPVersion []struct {
				Protocol string `json:"clientHTTPProtocol"`
				Requests uint64 `json:"requests"`
			} `json:"clientHTTPVersionMap"`
			ClientSSL []struct {
				Protocol string `json:"clientSSLProtocol"`
			} `json:"clientSSLMap"`
			ContentType []struct {
				Bytes                   uint64 `json:"bytes"`
				Requests                uint64 `json:"requests"`
				EdgeResponseContentType string `json:"edgeResponseContentTypeName"`
			} `json:"contentTypeMap"`
			Country []struct {
				Bytes             uint64 `json:"bytes"`
				ClientCountryName string `json:"clientCountryName"`
				Requests          uint64 `json:"requests"`
				Threats           uint64 `json:"threats"`
			} `json:"countryMap"`
			EncryptedBytes    uint64 `json:"encryptedBytes"`
			EncryptedRequests uint64 `json:"encryptedRequests"`
			IPClass           []struct {
				Type     string `json:"ipType"`
				Requests uint64 `json:"requests"`
			} `json:"ipClassMap"`
			PageViews      uint64 `json:"pageViews"`
			ResponseStatus []struct {
				EdgeResponseStatus int    `json:"edgeResponseStatus"`
				Requests           uint64 `json:"requests"`
			} `json:"responseStatusMap"`
			ThreatPathing []struct {
				Name     string `json:"threatPathingName"`
				Requests uint64 `json:"requests	"`
			} `json:"threatPathingMap"`
			Threats uint64 `json:"threats"`
		} `json:"sum"`
	} `json:"httpRequests1mGroups"`

	ColoGroups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
			ColoCode string `json:"coloCode"`
		} `json:"dimensions"`
		Count uint64 `json:"count"`
		Sum struct {
			EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
			Visits uint64 `json:"visits"`
		} `json:"sum"`
		Avg struct {
			sampleInterval uint64 `json:"sampleInterval"`
		} `json:"avg"`
	} `json:"httpRequestsAdaptiveGroups"`

	FirewallEventsAdaptiveGroups []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			Action                   string `json:"action"`
			ClientCountryName string `json:"clientCountryName"`
			ClientRequestHTTPHost    string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"firewallEventsAdaptiveGroups"`

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

func fetchZoneTotals(zoneIDs []string) (*cloudflareResponse, error) {
	now := time.Now().Add(-180 * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	http1mGroups := graphql.NewRequest(`
query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
	viewer {
		zones(filter: { zoneTag_in: $zoneIDs }) {
			zoneTag
			httpRequests1mGroups(limit: $limit filter: { datetime: $maxtime }) {
				uniq {
					uniques
				}
				sum {
					browserMap {
						pageViews
						uaBrowserFamily
					}
					bytes
					cachedBytes
					cachedRequests
					clientHTTPVersionMap {
						clientHTTPProtocol
						requests
					}
					clientSSLMap {
						clientSSLProtocol
						requests
					}
					contentTypeMap {
						bytes
						requests
						edgeResponseContentTypeName
					}
					countryMap {
						bytes
						clientCountryName
						requests
						threats
					}
					encryptedBytes
					encryptedRequests
					ipClassMap {
						ipType
						requests
					}
					pageViews
					requests
					responseStatusMap {
						edgeResponseStatus
						requests
					}
					threatPathingMap {
						requests
						threatPathingName
					}
					threats
				}
				dimensions {
					datetime
				}
			}
			firewallEventsAdaptiveGroups(limit: $limit, filter: {datetime_geq: $mintime, datetime_lt: $maxtime}) {
				count
				dimensions {
				  action	
				  source
				  clientRequestHTTPHost
				  clientCountryName
				}
			}
		}
	}
}
`)

	http1mGroups.Header.Set("X-AUTH-EMAIL", os.Getenv("CF_API_EMAIL"))
	http1mGroups.Header.Set("X-AUTH-KEY", os.Getenv("CF_API_KEY"))
	http1mGroups.Var("limit", 9999)
	http1mGroups.Var("maxtime", now)
	http1mGroups.Var("mintime", now1mAgo)
	http1mGroups.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient("https://api.cloudflare.com/client/v4/graphql/")

	var resp cloudflareResponse
	if err := graphqlClient.Run(ctx, http1mGroups, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchColoTotals(zoneIDs []string) (*cloudflareResponse, error) {

	now := time.Now().Add(time.Duration(-180) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)

	httpRequestsAdaptiveGroups := graphql.NewRequest(`
	query ($zoneIDs: [String!], $time: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				httpRequestsAdaptiveGroups(
					limit: $limit
					filter: { datetime: $time }
					) {
						count
						avg {
							sampleInterval
						}
						dimensions {
							coloCode
							datetime
						}
						sum {
							edgeResponseBytes
							visits
						}
					}
				}
			}
		}
`)

	httpRequestsAdaptiveGroups.Header.Set("X-AUTH-EMAIL", os.Getenv("CF_API_EMAIL"))
	httpRequestsAdaptiveGroups.Header.Set("X-AUTH-KEY", os.Getenv("CF_API_KEY"))
	httpRequestsAdaptiveGroups.Var("limit", 9999)
	httpRequestsAdaptiveGroups.Var("time", now)
	httpRequestsAdaptiveGroups.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient("https://api.cloudflare.com/client/v4/graphql/")
	var resp cloudflareResponse
	if err := graphqlClient.Run(ctx, httpRequestsAdaptiveGroups, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func findZoneName(zones []cloudflare.Zone, ID string) string {
	for _, z := range zones {
		if z.ID == ID {
			return z.Name
		}
	}

	return ""
}

func extractZoneIDs(zones []cloudflare.Zone) []string {
	var IDs []string

	for _, z := range zones {
		IDs = append(IDs, z.ID)
	}

	return IDs
}
