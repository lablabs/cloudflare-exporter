package main

import (
	"context"
	"strconv"

	"github.com/biter777/countries"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/machinebox/graphql"
	"github.com/prometheus/client_golang/prometheus"
)

// cloudflareResponseFree is the response shape for the free-tier zone
// analytics query. It is built entirely on httpRequestsAdaptiveGroups, the
// only HTTP analytics dataset enabled on the Free plan (httpRequests1mGroups,
// firewallEventsAdaptiveGroups and healthCheckEventsAdaptiveGroups are not).
type cloudflareResponseFree struct {
	Viewer struct {
		Zones []zoneRespFree `json:"zones"`
	} `json:"viewer"`
}

type zoneRespFree struct {
	ZoneTag                    string `json:"zoneTag"`
	HTTPRequestsAdaptiveGroups []struct {
		Count uint64 `json:"count"`
		Sum   struct {
			EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
		} `json:"sum"`
		Dimensions struct {
			ClientCountryName     string `json:"clientCountryName"`
			EdgeResponseStatus    uint16 `json:"edgeResponseStatus"`
			CacheStatus           string `json:"cacheStatus"`
			ClientSSLProtocol     string `json:"clientSSLProtocol"`
			ClientRequestHTTPHost string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"httpRequestsAdaptiveGroups"`
}

// cacheHitStatuses lists the cacheStatus values that count as a cache hit.
// "ignored" is deliberately excluded: it means the response was not
// cache-eligible (e.g. origin Cache-Control), not that it was served from
// cache. See https://developers.cloudflare.com/analytics/graphql-api/ for
// the current enum.
var cacheHitStatuses = map[string]struct{}{
	"hit":             {},
	"stale":           {},
	"updating":        {},
	"revalidated":     {},
	"deferred_hit":    {},
	"revalidated_hit": {},
}

func isCacheHit(cacheStatus string) bool {
	_, ok := cacheHitStatuses[cacheStatus]
	return ok
}

func isSSLEncrypted(clientSSLProtocol string) bool {
	return clientSSLProtocol != "" && clientSSLProtocol != "none"
}

func fetchZoneTotalsFree(zoneIDs []string) (*cloudflareResponseFree, error) {
	request := graphql.NewRequest(`
	query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				httpRequestsAdaptiveGroups(
					limit: $limit
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime, requestSource_in: ["eyeball"] }
				) {
					count
					sum {
						edgeResponseBytes
					}
					dimensions {
						clientCountryName
						edgeResponseStatus
						cacheStatus
						clientSSLProtocol
						clientRequestHTTPHost
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

	var resp cloudflareResponseFree
	if err := gql.Client.Run(ctx, request, &resp); err != nil {
		log.Errorf("failed to fetch free-tier zone analytics, err:%v", err)
		return nil, err
	}

	return &resp, nil
}

// fetchZoneAnalyticsFree is the free-tier counterpart of fetchZoneAnalytics.
// It is a plain function call (not a goroutine) invoked from within
// fetchZoneAnalytics before that function's own deferred wg.Done() fires, so
// it must not touch the caller's WaitGroup.
func fetchZoneAnalyticsFree(zones []cfzones.Zone) {
	zoneIDs := extractZoneIDs(zones)
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchZoneTotalsFree(zoneIDs)
	if err != nil {
		log.Error("failed to fetch free-tier zone analytics: ", err)
		return
	}

	for _, z := range r.Viewer.Zones {
		name, account := findZoneAccountName(zones, z.ZoneTag)
		if len(z.HTTPRequestsAdaptiveGroups) >= gqlQueryLimit {
			log.Warnf("free-tier zone analytics for zone %s hit the %d-row query limit; results may be truncated because the query groups by 5 dimensions (country, status, cacheStatus, sslProtocol, host)", name, gqlQueryLimit)
		}
		z := z
		addAdaptiveHTTPGroups(&z, name, account)
	}
}

// addAdaptiveHTTPGroups maps a free-tier httpRequestsAdaptiveGroups response
// onto the same Prometheus series addHTTPGroups populates from the paid-tier
// httpRequests1mGroups response, so existing dashboards keep working
// unmodified. Metrics that have no equivalent in this dataset (uniques,
// pageviews, browser map, threats, firewall events, health checks) are left
// untouched here; see README for the full list.
func addAdaptiveHTTPGroups(z *zoneRespFree, name string, account string) {
	if len(z.HTTPRequestsAdaptiveGroups) == 0 {
		return
	}

	// Clear stale series for this zone/account across the metrics populated below.
	label := prometheus.Labels{"zone": name, "account": account}
	zoneRequestTotal.DeletePartialMatch(label)
	zoneRequestCached.DeletePartialMatch(label)
	zoneRequestSSLEncrypted.DeletePartialMatch(label)
	zoneRequestContentType.DeletePartialMatch(label)
	zoneBandwidthContentType.DeletePartialMatch(label)
	zoneRequestCountry.DeletePartialMatch(label)
	zoneBandwidthCountry.DeletePartialMatch(label)
	zoneRequestHTTPStatus.DeletePartialMatch(label)
	zoneBandwidthTotal.DeletePartialMatch(label)
	zoneBandwidthCached.DeletePartialMatch(label)
	zoneBandwidthSSLEncrypted.DeletePartialMatch(label)
	zoneRequestStatusCountryHost.DeletePartialMatch(label)

	var requestTotal, requestCached, requestSSL float64
	var bandwidthTotal, bandwidthCached, bandwidthSSL float64

	for _, g := range z.HTTPRequestsAdaptiveGroups {
		count := float64(g.Count)
		bytes := float64(g.Sum.EdgeResponseBytes)

		requestTotal += count
		bandwidthTotal += bytes

		if isCacheHit(g.Dimensions.CacheStatus) {
			requestCached += count
			bandwidthCached += bytes
		}

		if isSSLEncrypted(g.Dimensions.ClientSSLProtocol) {
			requestSSL += count
			bandwidthSSL += bytes
		}

		if g.Dimensions.ClientCountryName != "" {
			c := countries.ByName(g.Dimensions.ClientCountryName)
			region := c.Info().Region.Info().Name
			zoneRequestCountry.With(prometheus.Labels{"zone": name, "account": account, "country": g.Dimensions.ClientCountryName, "region": region}).Add(count)
			zoneBandwidthCountry.With(prometheus.Labels{"zone": name, "account": account, "country": g.Dimensions.ClientCountryName, "region": region}).Add(bytes)
		}

		status := strconv.Itoa(int(g.Dimensions.EdgeResponseStatus))
		zoneRequestHTTPStatus.With(prometheus.Labels{"zone": name, "account": account, "status": status}).Add(count)

		zoneRequestStatusCountryHost.With(
			prometheus.Labels{
				"zone":    name,
				"account": account,
				"status":  status,
				"country": g.Dimensions.ClientCountryName,
				"host":    g.Dimensions.ClientRequestHTTPHost,
			}).Add(count)
	}

	zoneRequestTotal.With(prometheus.Labels{"zone": name, "account": account}).Add(requestTotal)
	zoneRequestCached.With(prometheus.Labels{"zone": name, "account": account}).Add(requestCached)
	zoneRequestSSLEncrypted.With(prometheus.Labels{"zone": name, "account": account}).Add(requestSSL)

	zoneBandwidthTotal.With(prometheus.Labels{"zone": name, "account": account}).Add(bandwidthTotal)
	zoneBandwidthCached.With(prometheus.Labels{"zone": name, "account": account}).Add(bandwidthCached)
	zoneBandwidthSSLEncrypted.With(prometheus.Labels{"zone": name, "account": account}).Add(bandwidthSSL)
}
