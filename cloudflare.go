package main

import (
	"context"
	"slices"
	"strings"

	cf "github.com/cloudflare/cloudflare-go/v4"
	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	cfload_balancers "github.com/cloudflare/cloudflare-go/v4/load_balancers"
	cfpagination "github.com/cloudflare/cloudflare-go/v4/packages/pagination"
	cfrulesets "github.com/cloudflare/cloudflare-go/v4/rulesets"
	cfzero_trust "github.com/cloudflare/cloudflare-go/v4/zero_trust"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"

	"github.com/machinebox/graphql"
)

const (
	freePlanID      = "0feeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	apiPerPageLimit = 999
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

type cloudflareResponseLogpushAccount struct {
	Viewer struct {
		Accounts []logpushResponse `json:"accounts"`
	} `json:"viewer"`
}

type r2AccountResp struct {
	R2StorageGroups []struct {
		Dimensions struct {
			BucketName string `json:"bucketName"`
		} `json:"dimensions"`
		Max struct {
			MetadataSize uint64 `json:"metadataSize"`
			PayloadSize  uint64 `json:"payloadSize"`
			ObjectCount  uint64 `json:"objectCount"`
		} `json:"max"`
	} `json:"r2StorageAdaptiveGroups"`

	R2StorageOperations []struct {
		Dimensions struct {
			Action     string `json:"actionType"`
			BucketName string `json:"bucketName"`
		} `json:"dimensions"`
		Sum struct {
			Requests uint64 `json:"requests"`
		} `json:"sum"`
	} `json:"r2OperationsAdaptiveGroups"`
}

type cloudflareResponseR2Account struct {
	Viewer struct {
		Accounts []r2AccountResp `json:"accounts"`
	}
}

type cloudflareResponseLogpushZone struct {
	Viewer struct {
		Zones []logpushResponse `json:"zones"`
	} `json:"viewer"`
}

type logpushResponse struct {
	LogpushHealthAdaptiveGroups []struct {
		Count uint64 `json:"count"`

		Dimensions struct {
			Datetime        string `json:"datetime"`
			DestinationType string `json:"destinationType"`
			JobID           int    `json:"jobId"`
			Status          int    `json:"status"`
			Final           int    `json:"final"`
		}
	} `json:"logpushHealthAdaptiveGroups"`
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
			RuleID                string `json:"ruleId"`
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

func fetchLoadblancerPools(account cfaccounts.Account) []cfload_balancers.Pool {
	var cfPools []cfload_balancers.Pool
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page := cfclient.LoadBalancers.Pools.ListAutoPaging(ctx,
		cfload_balancers.PoolListParams{
			AccountID: cf.F(account.ID),
		})
	if page.Err() != nil {
		log.Errorf("error fetching loadbalancer pools, err:%v", page.Err())
		return nil
	}

	seenIDs := make(map[string]struct{})
	for page.Next() {
		if page.Err() != nil {
			log.Errorf("error during paging pools: %v", page.Err())
			break
		}
		pool := page.Current()
		if _, exists := seenIDs[pool.ID]; exists {
			log.Errorf("fetchLoadbalancerPools: duplicate pool ID detected (%s), breaking loop", pool.ID)
			break
		}
		seenIDs[pool.ID] = struct{}{}
		cfPools = append(cfPools, pool)
	}

	return cfPools
}

func getAccountZoneList(accountID string) ([]cfzones.Zone, error) {
	var zoneList []cfzones.Zone
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page := cfclient.Zones.ListAutoPaging(ctx, cfzones.ZoneListParams{
		Account: cf.F(cfzones.ZoneListParamsAccount{ID: cf.F(accountID)}),
		PerPage: cf.F(float64(apiPerPageLimit)),
	})
	if page.Err() != nil {
		return nil, page.Err()
	}

	seenIDs := make(map[string]struct{})
	for page.Next() {
		if page.Err() != nil {
			log.Errorf("error during paging zoneList: %v", page.Err())
			break
		}
		zone := page.Current()
		if _, exists := seenIDs[zone.ID]; exists {
			log.Errorf("getAccountZoneList: duplicate zone ID detected (%s), breaking loop", zone.ID)
			break
		}
		seenIDs[zone.ID] = struct{}{}
		zoneList = append(zoneList, zone)
	}

	return zoneList, nil
}

func fetchZones(accounts []cfaccounts.Account) []cfzones.Zone {
	var zones []cfzones.Zone

	for _, account := range accounts {
		z, err := getAccountZoneList(account.ID)

		if err != nil {
			log.Errorf("error fetching zones: %v", err)
			continue
		}
		zones = append(zones, z...)
	}
	return zones
}

func getRuleSetsList(params cfrulesets.RulesetListParams) ([]cfrulesets.RulesetListResponse, error) {
	var ruleSetList []cfrulesets.RulesetListResponse
	var page *cfpagination.CursorPagination[cfrulesets.RulesetListResponse]
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page, err = cfclient.Rulesets.List(ctx, params)
	if err != nil {
		return nil, err
	}

	ruleSetList = append(ruleSetList, page.Result...)

	for page.ResultInfo.Cursor != "" {
		params.Cursor = cf.F(page.ResultInfo.Cursor)
		ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
		page, err = cfclient.Rulesets.List(ctx, params)
		cancel()
		if err != nil {
			return nil, err
		}
		ruleSetList = append(ruleSetList, page.Result...)
	}

	return ruleSetList, nil
}

func fetchFirewallRules(zoneID string) map[string]string {
	listOfRulesets, err := getRuleSetsList(cfrulesets.RulesetListParams{
		ZoneID: cf.F(zoneID),
	})
	if err != nil {
		log.Errorf("error fetching firewall rules, ZoneID:%s, Err:%v", zoneID, err)
		return nil
	}

	firewallRulesMap := make(map[string]string)

	for _, rulesetDesc := range listOfRulesets {
		if rulesetDesc.Phase == cfrulesets.PhaseHTTPRequestFirewallManaged {
			ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
			ruleset, err := cfclient.Rulesets.Get(ctx, rulesetDesc.ID, cfrulesets.RulesetGetParams{
				ZoneID: cf.F(zoneID),
			})
			if err != nil {
				log.Errorf("error fetching ruleset for managed firewall rules, ZoneID:%s, RulesetID:%s, Err:%v", zoneID, rulesetDesc.ID, err)
				cancel()
				continue
			}
			cancel()
			for _, rule := range ruleset.Rules {
				firewallRulesMap[rule.ID] = rule.Description
			}
		}

		if rulesetDesc.Phase == cfrulesets.PhaseHTTPRequestFirewallCustom {
			ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
			ruleset, err := cfclient.Rulesets.Get(ctx, rulesetDesc.ID, cfrulesets.RulesetGetParams{
				ZoneID: cf.F(zoneID),
			})
			if err != nil {
				log.Errorf("error fetching ruleset for custom firewall rules, ZoneID:%s, RulesetID:%s, Err:%v", zoneID, rulesetDesc.ID, err)
				cancel()
				continue
			}
			cancel()
			for _, rule := range ruleset.Rules {
				firewallRulesMap[rule.ID] = rule.Description
			}
		}
	}

	return firewallRulesMap
}

func fetchAccounts() []cfaccounts.Account {
	var cfAccounts []cfaccounts.Account
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page := cfclient.Accounts.ListAutoPaging(ctx,
		cfaccounts.AccountListParams{
			PerPage: cf.F(float64(apiPerPageLimit)),
		})
	if page.Err() != nil {
		log.Errorf("error fetching accounts:%v", page.Err())
		return nil
	}

	seenIDs := make(map[string]struct{})
	for page.Next() {
		if page.Err() != nil {
			log.Errorf("error during paging accounts: %v", page.Err())
			break
		}
		account := page.Current()
		if _, exists := seenIDs[account.ID]; exists {
			log.Errorf("fetchAccounts: duplicate account ID detected (%s), breaking loop", account.ID)
			break
		}
		seenIDs[account.ID] = struct{}{}
		cfAccounts = append(cfAccounts, account)
	}
	return cfAccounts
}

func fetchZoneTotals(zoneIDs []string) (*cloudflareResponse, error) {
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
			httpRequestsEdgeCountryHost: httpRequestsAdaptiveGroups(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime, requestSource_in: ["eyeball"] }) {
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

	now, now1mAgo := GetTimeRange()
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponse
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("failed to fetch zone totals, err:%v", err)
		return nil, err
	}

	return &resp, nil
}

func fetchColoTotals(zoneIDs []string) (*cloudflareResponseColo, error) {
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

	now, now1mAgo := GetTimeRange()
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponseColo
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("failed to fetch colocation totals, err:%v", err)
		return nil, err
	}

	return &resp, nil
}

func fetchWorkerTotals(accountID string) (*cloudflareResponseAccts, error) {
	request := graphql.NewRequest(`
	query ($accountID: String!, $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			accounts(filter: {accountTag: $accountID} ) {
				workersInvocationsAdaptive(limit: $limit, filter: { datetime_geq: $mintime, datetime_lt: $maxtime}) {
					dimensions {
						scriptName
						status
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

	now, now1mAgo := GetTimeRange()
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("accountID", accountID)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponseAccts
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("error fetching worker totals, err:%v", err)
		return nil, err
	}

	return &resp, nil
}

func fetchLoadBalancerTotals(zoneIDs []string) (*cloudflareResponseLb, error) {
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

	now, now1mAgo := GetTimeRange()
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponseLb
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("error fetching load balancer totals, err:%v", err)
		return nil, err
	}
	return &resp, nil
}

func fetchLogpushAccount(accountID string) (*cloudflareResponseLogpushAccount, error) {
	request := graphql.NewRequest(`query($accountID: String!, $limit: Int!, $mintime: Time!, $maxtime: Time!) {
		viewer {
		  accounts(filter: {accountTag : $accountID }) {
			logpushHealthAdaptiveGroups(
			  filter: {
				datetime_geq: $mintime
				datetime_lt: $maxtime
				status_neq: 200
			  }
			  limit: $limit
			) {
			  count
			  dimensions {
				jobId
				status
				destinationType
				datetime
				final
			  }
			}
		  }
		}
	  }`)

	now, now1mAgo := GetTimeRange()
	request.Var("accountID", accountID)
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	var resp cloudflareResponseLogpushAccount
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("error fetching logpush account totals, err:%v", err)
		return nil, err
	}
	return &resp, nil
}

func fetchLogpushZone(zoneIDs []string) (*cloudflareResponseLogpushZone, error) {
	request := graphql.NewRequest(`query($zoneIDs: String!, $limit: Int!, $mintime: Time!, $maxtime: Time!) {
		viewer {
			zones(filter: {zoneTag_in : $zoneIDs }) {
			logpushHealthAdaptiveGroups(
			  filter: {
				datetime_geq: $mintime
				datetime_lt: $maxtime
				status_neq: 200
			  }
			  limit: $limit
			) {
			  count
			  dimensions {
				jobId
				status
				destinationType
				datetime
				final
			  }
			}
		  }
		}
	  }`)

	now, now1mAgo := GetTimeRange()
	request.Var("zoneIDs", zoneIDs)
	request.Var("limit", gqlQueryLimit)
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponseLogpushZone
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("error fetching logpush zone totals, err:%v", err)
		return nil, err
	}

	return &resp, nil
}

func fetchR2Account(accountID string) (*cloudflareResponseR2Account, error) {
	request := graphql.NewRequest(`query($accountID: String!, $limit: Int!, $date: String!) {
		viewer {
		  accounts(filter: {accountTag : $accountID }) {
			r2StorageAdaptiveGroups(
			  filter: {
				date: $date
			  },
			  limit: $limit
			) {
			  dimensions {
          		bucketName
			  }
        	  max {
				metadataSize
          		payloadSize
				objectCount
			  }
      		}
			r2OperationsAdaptiveGroups(filter: { date: $date }, limit: $limit) {
				dimensions {
					actionType
					bucketName
				}
				sum {
					requests
				}
			}
			}
		  }
	  }`)

	now, _ := GetTimeRange()
	request.Var("accountID", accountID)
	request.Var("limit", gqlQueryLimit)
	request.Var("date", now.Format("2006-01-02"))

	gql.Mu.RLock()
	defer gql.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()

	var resp cloudflareResponseR2Account
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("error fetching R2 account: %v", err)
		return nil, err
	}
	return &resp, nil
}

func fetchCloudflareTunnels(account cfaccounts.Account) []cfzero_trust.TunnelListResponse {
	var cfTunnels []cfzero_trust.TunnelListResponse
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page := cfclient.ZeroTrust.Tunnels.ListAutoPaging(ctx,
		cfzero_trust.TunnelListParams{
			AccountID: cf.F(account.ID),
			PerPage:   cf.F(float64(apiPerPageLimit)),
			IsDeleted: cf.F(false),
		})
	if page.Err() != nil {
		log.Errorf("error fetching tunnels, err:%v", page.Err())
		return nil
	}

	seenIDs := make(map[string]struct{})
	for page.Next() {
		if page.Err() != nil {
			log.Errorf("error during paging tunnels: %v", page.Err())
			break
		}
		tunnel := page.Current()
		if _, exists := seenIDs[tunnel.ID]; exists {
			log.Errorf("fetchCloudflareTunnels: duplicate tunnel ID detected (%s), breaking loop", tunnel.ID)
			break
		}
		seenIDs[tunnel.ID] = struct{}{}
		cfTunnels = append(cfTunnels, tunnel)
	}

	return cfTunnels
}

func fetchCloudflareTunnelConnectors(account cfaccounts.Account, tunnelID string) []cfzero_trust.Client {
	var cfClients []cfzero_trust.Client
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	page := cfclient.ZeroTrust.Tunnels.Connections.GetAutoPaging(ctx,
		tunnelID,
		cfzero_trust.TunnelConnectionGetParams{
			AccountID: cf.F(account.ID),
		})
	if page.Err() != nil {
		log.Errorf("error fetching tunnel connections, err:%v", page.Err())
		return nil
	}

	for page.Next() {
		if page.Err() != nil {
			log.Errorf("error during paging tunnel connections: %v", page.Err())
			break
		}
		client := page.Current()
		cfClients = append(cfClients, client)
	}

	return cfClients
}

func findZoneAccountName(zones []cfzones.Zone, ID string) (string, string) {
	for _, z := range zones {
		if z.ID == ID {
			return z.Name, strings.ToLower(strings.ReplaceAll(z.Account.Name, " ", "-"))
		}
	}

	return "", ""
}

func extractZoneIDs(zones []cfzones.Zone) []string {
	var IDs []string

	for _, z := range zones {
		IDs = append(IDs, z.ID)
	}

	return IDs
}

func filterNonFreePlanZones(zones []cfzones.Zone) (filteredZones []cfzones.Zone) {
	var zoneIDs []string

	for _, z := range zones {
		extraFields, err := jsonStringToMap(z.JSON.ExtraFields["plan"].Raw())
		if err != nil {
			log.Error(err)
			continue
		}
		if extraFields["id"] == freePlanID {
			continue
		}
		if !slices.Contains(zoneIDs, z.ID) {
			zoneIDs = append(zoneIDs, z.ID)
			filteredZones = append(filteredZones, z)
		}
	}
	return
}
