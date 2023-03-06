package main

import (
	"context"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

var (
	cfGraphQLEndpoint = "https://api.cloudflare.com/client/v4/graphql/"
)

type cloudflareResponse struct {
	Viewer struct {
		Zones []zoneResp `json:"zones"`
	} `json:"viewer"`
}

type cloudflareResponseAccts struct {
	Viewer struct {
		Accounts []accountResp `json:"accounts"`
	} `json:"viewer"`
}

type cloudflareResponseColo struct {
	Viewer struct {
		Zones []zoneRespColo `json:"zones"`
	} `json:"viewer"`
}

type cloudflareResponseLb struct {
	Viewer struct {
		Zones []lbResp `json:"zones"`
	} `json:"viewer"`
}

type accountResp struct {
	WorkersInvocationsAdaptive []struct {
		Dimensions struct {
			ScriptName string `json:"scriptName"`
			Status     string `json:"status"`
		}

		Sum struct {
			Requests uint64  `json:"requests"`
			Errors   uint64  `json:"errors"`
			Duration float64 `json:"duration"`
		} `json:"sum"`

		Quantiles struct {
			CPUTimeP50   float32 `json:"cpuTimeP50"`
			CPUTimeP75   float32 `json:"cpuTimeP75"`
			CPUTimeP99   float32 `json:"cpuTimeP99"`
			CPUTimeP999  float32 `json:"cpuTimeP999"`
			DurationP50  float32 `json:"durationP50"`
			DurationP75  float32 `json:"durationP75"`
			DurationP99  float32 `json:"durationP99"`
			DurationP999 float32 `json:"durationP999"`
		} `json:"quantiles"`
	} `json:"workersInvocationsAdaptive"`
}

type zoneRespColo struct {
	ColoGroups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
			ColoCode string `json:"coloCode"`
			Host     string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
		Count uint64 `json:"count"`
		Sum   struct {
			EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
			Visits            uint64 `json:"visits"`
		} `json:"sum"`
		Avg struct {
			SampleInterval float64 `json:"sampleInterval"`
		} `json:"avg"`
	} `json:"httpRequestsAdaptiveGroups"`

	ZoneTag string `json:"zoneTag"`
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
			BrowserMap     []struct {
				PageViews       uint64 `json:"pageViews"`
				UaBrowserFamily string `json:"uaBrowserFamily"`
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
				Requests uint64 `json:"requests"`
			} `json:"threatPathingMap"`
			Threats uint64 `json:"threats"`
		} `json:"sum"`
	} `json:"httpRequests1mGroups"`

	FirewallEventsAdaptiveGroups []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			Action                string `json:"action"`
			Source                string `json:"source"`
			RuleId                string `json:"ruleId"`
			ClientCountryName     string `json:"clientCountryName"`
			ClientRequestHTTPHost string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"firewallEventsAdaptiveGroups"`

	HTTPRequestsAdaptiveGroups []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			OriginResponseStatus  uint16 `json:"originResponseStatus"`
			ClientCountryName     string `json:"clientCountryName"`
			ClientRequestHTTPHost string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"httpRequestsAdaptiveGroups"`

	HTTPRequestsEdgeCountryHost []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			EdgeResponseStatus    uint16 `json:"edgeResponseStatus"`
			ClientCountryName     string `json:"clientCountryName"`
			ClientRequestHTTPHost string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"httpRequestsEdgeCountryHost"`

	HealthCheckEventsAdaptiveGroups []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			HealthStatus  string `json:"healthStatus"`
			OriginIP      string `json:"originIP"`
			FailureReason string `json:"failureReason"`
			Region        string `json:"region"`
			Fqdn          string `json:"fqdn"`
		} `json:"dimensions"`
	} `json:"healthCheckEventsAdaptiveGroups"`

	ZoneTag string `json:"zoneTag"`
}

type lbResp struct {
	LoadBalancingRequestsAdaptiveGroups []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			LbName               string `json:"lbName"`
			Proxied              uint8  `json:"proxied"`
			Region               string `json:"region"`
			SelectedOriginName   string `json:"selectedOriginName"`
			SelectedPoolAvgRttMs uint64 `json:"selectedPoolAvgRttMs"`
			SelectedPoolHealthy  uint8  `json:"selectedPoolHealthy"`
			SelectedPoolName     string `json:"selectedPoolName"`
			SteeringPolicy       string `json:"steeringPolicy"`
		} `json:"dimensions"`
	} `json:"loadBalancingRequestsAdaptiveGroups"`

	LoadBalancingRequestsAdaptive []struct {
		LbName                string `json:"lbName"`
		Proxied               uint8  `json:"proxied"`
		Region                string `json:"region"`
		SelectedPoolHealthy   uint8  `json:"selectedPoolHealthy"`
		SelectedPoolID        string `json:"selectedPoolID"`
		SelectedPoolName      string `json:"selectedPoolName"`
		SessionAffinityStatus string `json:"sessionAffinityStatus"`
		SteeringPolicy        string `json:"steeringPolicy"`
		SelectedPoolAvgRttMs  uint64 `json:"selectedPoolAvgRttMs"`
		Pools                 []struct {
			AvgRttMs uint64 `json:"avgRttMs"`
			Healthy  uint8  `json:"healthy"`
			ID       string `json:"id"`
			PoolName string `json:"poolName"`
		} `json:"pools"`
		Origins []struct {
			OriginName string `json:"originName"`
			Health     uint8  `json:"health"`
			IPv4       string `json:"ipv4"`
			Selected   uint8  `json:"selected"`
		} `json:"origins"`
	} `json:"loadBalancingRequestsAdaptive"`

	ZoneTag string `json:"zoneTag"`
}

func fetchZones() []cloudflare.Zone {
	var api *cloudflare.API
	var err error
	if len(cfgCfAPIToken) > 0 {
		api, err = cloudflare.NewWithAPIToken(cfgCfAPIToken)
	} else {
		api, err = cloudflare.New(cfgCfAPIKey, cfgCfAPIEmail)
	}
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	z, err := api.ListZones(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return z
}

func fetchFirewallRules(zoneId string) map[string]string {
	var api *cloudflare.API
	var err error
	if len(cfgCfAPIToken) > 0 {
		api, err = cloudflare.NewWithAPIToken(cfgCfAPIToken)
	} else {
		api, err = cloudflare.New(cfgCfAPIKey, cfgCfAPIEmail)
	}
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	listOfRules, _, err := api.FirewallRules(ctx,
		cloudflare.ZoneIdentifier(zoneId),
		cloudflare.FirewallRuleListParams{})
	if err != nil {
		log.Fatal(err)
	}
	firewallRulesMap := make(map[string]string)

	for _, rule := range listOfRules {
		firewallRulesMap[rule.ID] = rule.Description
	}
	return firewallRulesMap
}

func fetchAccounts() []cloudflare.Account {
	var api *cloudflare.API
	var err error
	if len(cfgCfAPIToken) > 0 {
		api, err = cloudflare.NewWithAPIToken(cfgCfAPIToken)
	} else {
		api, err = cloudflare.New(cfgCfAPIKey, cfgCfAPIEmail)
	}
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	a, _, err := api.Accounts(ctx, cloudflare.AccountsListParams{PaginationOptions: cloudflare.PaginationOptions{PerPage: 100}})
	if err != nil {
		log.Fatal(err)
	}

	return a
}

func fetchZoneTotals(zoneIDs []string) (*cloudflareResponse, error) {
	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	request := graphql.NewRequest(`
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
			firewallEventsAdaptiveGroups(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime }) {
				count
				dimensions {
				  action
				  source
				  ruleId
				  clientRequestHTTPHost
				  clientCountryName
				}
			}
			httpRequestsAdaptiveGroups(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime, cacheStatus_notin: ["hit"] }) {
				count
				dimensions {
					originResponseStatus
					clientCountryName
					clientRequestHTTPHost
				}
			}
			httpRequestsEdgeCountryHost: httpRequestsAdaptiveGroups(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime }) {
				count
				dimensions {
					edgeResponseStatus
					clientCountryName
					clientRequestHTTPHost
				}
			}
			healthCheckEventsAdaptiveGroups(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime }) {
				count
				dimensions {
					healthStatus
					originIP
					region
					fqdn
				}
			}
		}
	}
}
`)
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)

	var resp cloudflareResponse
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchColoTotals(zoneIDs []string) (*cloudflareResponseColo, error) {
	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	request := graphql.NewRequest(`
	query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				httpRequestsAdaptiveGroups(
					limit: $limit
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime }
					) {
						count
						avg {
							sampleInterval
						}
						dimensions {
							clientRequestHTTPHost
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
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)
	var resp cloudflareResponseColo
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchWorkerTotals(accountID string) (*cloudflareResponseAccts, error) {
	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	request := graphql.NewRequest(`
	query ($accountID: String!, $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			accounts(filter: {accountTag: $accountID} ) {
				workersInvocationsAdaptive(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime}) {
					dimensions {
						scriptName
						status
						datetime
					}

					sum {
						requests
						errors
						duration
					}

					quantiles {
						cpuTimeP50
						cpuTimeP75
						cpuTimeP99
						cpuTimeP999
						durationP50
						durationP75
						durationP99
						durationP999
					}
				}
			}
		}
	}
`)
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("accountID", accountID)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)
	var resp cloudflareResponseAccts
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchLoadBalancerTotals(zoneIDs []string) (*cloudflareResponseLb, error) {
	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	request := graphql.NewRequest(`
	query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				loadBalancingRequestsAdaptiveGroups(
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime},
					limit: $limit) {
					count
					dimensions {
						region
						lbName
						selectedPoolName
						proxied
						selectedOriginName
						selectedPoolAvgRttMs
						selectedPoolHealthy
						steeringPolicy
					}
				}
				loadBalancingRequestsAdaptive(
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime},
					limit: $limit) {
					lbName
					proxied
					region
					selectedPoolHealthy
					selectedPoolId
					selectedPoolName
					sessionAffinityStatus
					steeringPolicy
					selectedPoolAvgRttMs
					pools {
						id
						poolName
						healthy
						avgRttMs
					}
					origins {
						originName
						health
						ipv4
						selected
					}
				}
			}
		}
	}
`)
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)
	var resp cloudflareResponseLb
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
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

func filterNonFreePlanZones(zones []cloudflare.Zone) (filteredZones []cloudflare.Zone) {
	for _, z := range zones {
		if z.Plan.ZonePlanCommon.ID != "0feeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
			filteredZones = append(filteredZones, z)
		}
	}
	return
}
