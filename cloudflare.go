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
	Groups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
		} `json:"dimensions"`
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
		Sum struct {
			Bytes          uint64 `json:"bytes"`
			CachedBytes    uint64 `json:"cachedBytes"`
			CachedRequests uint64 `json:"cachedRequests"`
			Requests       uint64 `json:"requests"`
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

func fetchZoneTotals(zoneID string) (*cloudflareResponse, error) {
	now := time.Now().Add(time.Duration(-180) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)

	http1mGroups := graphql.NewRequest(`
query ($zoneID: String!, $time: Time!, $limit: Int!) {
	viewer {
		zones(filter: { zoneTag: $zoneID }) {
			zoneTag

			httpRequests1mGroups(
				limit: $limit
				filter: { datetime: $time }
			) {
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
		}
	}
}
`)

	http1mGroups.Header.Set("X-AUTH-EMAIL", os.Getenv("CF_API_EMAIL"))
	http1mGroups.Header.Set("X-AUTH-KEY", os.Getenv("CF_API_KEY"))
	http1mGroups.Var("limit", 9999)
	http1mGroups.Var("time", now)
	http1mGroups.Var("zoneID", zoneID)

	ctx := context.Background()
	graphqlClient := graphql.NewClient("https://api.cloudflare.com/client/v4/graphql/")

	var resp cloudflareResponse
	if err := graphqlClient.Run(ctx, http1mGroups, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchColoTotals(zoneID string) (*cloudflareResponse, error) {

	now := time.Now().Add(time.Duration(-180) * time.Second).UTC()
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
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}
