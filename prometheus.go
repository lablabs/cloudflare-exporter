package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/biter777/countries"
	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type MetricName string

func (mn MetricName) String() string {
	return string(mn)
}

const (
	zoneRequestTotalMetricName                   MetricName = "cloudflare_zone_requests_total"
	zoneRequestCachedMetricName                  MetricName = "cloudflare_zone_requests_cached"
	zoneRequestSSLEncryptedMetricName            MetricName = "cloudflare_zone_requests_ssl_encrypted"
	zoneRequestContentTypeMetricName             MetricName = "cloudflare_zone_requests_content_type"
	zoneRequestCountryMetricName                 MetricName = "cloudflare_zone_requests_country"
	zoneRequestHTTPStatusMetricName              MetricName = "cloudflare_zone_requests_status"
	zoneRequestBrowserMapMetricName              MetricName = "cloudflare_zone_requests_browser_map_page_views_count"
	zoneRequestOriginStatusCountryHostMetricName MetricName = "cloudflare_zone_requests_origin_status_country_host"
	zoneRequestStatusCountryHostMetricName       MetricName = "cloudflare_zone_requests_status_country_host"
	zoneBandwidthTotalMetricName                 MetricName = "cloudflare_zone_bandwidth_total"
	zoneBandwidthCachedMetricName                MetricName = "cloudflare_zone_bandwidth_cached"
	zoneBandwidthSSLEncryptedMetricName          MetricName = "cloudflare_zone_bandwidth_ssl_encrypted"
	zoneBandwidthContentTypeMetricName           MetricName = "cloudflare_zone_bandwidth_content_type"
	zoneBandwidthCountryMetricName               MetricName = "cloudflare_zone_bandwidth_country"
	zoneThreatsTotalMetricName                   MetricName = "cloudflare_zone_threats_total"
	zoneThreatsCountryMetricName                 MetricName = "cloudflare_zone_threats_country"
	zoneThreatsTypeMetricName                    MetricName = "cloudflare_zone_threats_type"
	zonePageviewsTotalMetricName                 MetricName = "cloudflare_zone_pageviews_total"
	zoneUniquesTotalMetricName                   MetricName = "cloudflare_zone_uniques_total"
	zoneColocationVisitsMetricName               MetricName = "cloudflare_zone_colocation_visits"
	zoneColocationEdgeResponseBytesMetricName    MetricName = "cloudflare_zone_colocation_edge_response_bytes"
	zoneColocationRequestsTotalMetricName        MetricName = "cloudflare_zone_colocation_requests_total"
	zoneFirewallEventsCountMetricName            MetricName = "cloudflare_zone_firewall_events_count"
	zoneHealthCheckEventsOriginCountMetricName   MetricName = "cloudflare_zone_health_check_events_origin_count"
	workerRequestsMetricName                     MetricName = "cloudflare_worker_requests_count"
	workerErrorsMetricName                       MetricName = "cloudflare_worker_errors_count"
	workerCPUTimeMetricName                      MetricName = "cloudflare_worker_cpu_time"
	workerDurationMetricName                     MetricName = "cloudflare_worker_duration"
	poolHealthStatusMetricName                   MetricName = "cloudflare_zone_pool_health_status"
	poolRequestsTotalMetricName                  MetricName = "cloudflare_zone_pool_requests_total"
	poolOriginHealthStatusMetricName             MetricName = "cloudflare_pool_origin_health_status"
	logpushFailedJobsAccountMetricName           MetricName = "cloudflare_logpush_failed_jobs_account_count"
	logpushFailedJobsZoneMetricName              MetricName = "cloudflare_logpush_failed_jobs_zone_count"
	r2StorageTotalMetricName                     MetricName = "cloudflare_r2_storage_total_bytes"
	r2StorageMetricName                          MetricName = "cloudflare_r2_storage_bytes"
	r2OperationMetricName                        MetricName = "cloudflare_r2_operation_count"
	tunnelInfoMetricName                         MetricName = "cloudflare_tunnel_info"
	tunnelHealthStatusMetricName                 MetricName = "cloudflare_tunnel_health_status"
	tunnelConnectorInfoMetricName                MetricName = "cloudflare_tunnel_connector_info"
	tunnelConnectorActiveConnectionsMetricName   MetricName = "cloudflare_tunnel_connector_active_connections"
)

type MetricsMap map[MetricName]prometheus.Collector

var (
	// Requests
	zoneRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestTotalMetricName.String(),
		Help: "Number of requests for zone",
	}, []string{"zone", "account"},
	)

	zoneRequestCached = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestCachedMetricName.String(),
		Help: "Number of cached requests for zone",
	}, []string{"zone", "account"},
	)

	zoneRequestSSLEncrypted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestSSLEncryptedMetricName.String(),
		Help: "Number of encrypted requests for zone",
	}, []string{"zone", "account"},
	)

	zoneRequestContentType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestContentTypeMetricName.String(),
		Help: "Number of request for zone per content type",
	}, []string{"zone", "account", "content_type"},
	)

	zoneRequestCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestCountryMetricName.String(),
		Help: "Number of request for zone per country",
	}, []string{"zone", "account", "country", "region"},
	)

	zoneRequestHTTPStatus = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestHTTPStatusMetricName.String(),
		Help: "Number of request for zone per HTTP status",
	}, []string{"zone", "account", "status"},
	)

	zoneRequestBrowserMap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestBrowserMapMetricName.String(),
		Help: "Number of successful requests for HTML pages per zone",
	}, []string{"zone", "account", "family"},
	)

	zoneRequestOriginStatusCountryHost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestOriginStatusCountryHostMetricName.String(),
		Help: "Count of not cached requests for zone per origin HTTP status per country per host",
	}, []string{"zone", "account", "status", "country", "host"},
	)

	zoneRequestStatusCountryHost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestStatusCountryHostMetricName.String(),
		Help: "Count of requests for zone per edge HTTP status per country per host",
	}, []string{"zone", "account", "status", "country", "host"},
	)

	zoneBandwidthTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthTotalMetricName.String(),
		Help: "Total bandwidth per zone in bytes",
	}, []string{"zone", "account"},
	)

	zoneBandwidthCached = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthCachedMetricName.String(),
		Help: "Cached bandwidth per zone in bytes",
	}, []string{"zone", "account"},
	)

	zoneBandwidthSSLEncrypted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthSSLEncryptedMetricName.String(),
		Help: "Encrypted bandwidth per zone in bytes",
	}, []string{"zone", "account"},
	)

	zoneBandwidthContentType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthContentTypeMetricName.String(),
		Help: "Bandwidth per zone per content type",
	}, []string{"zone", "account", "content_type"},
	)

	zoneBandwidthCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneBandwidthCountryMetricName.String(),
		Help: "Bandwidth per country per zone",
	}, []string{"zone", "account", "country", "region"},
	)

	zoneThreatsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsTotalMetricName.String(),
		Help: "Threats per zone",
	}, []string{"zone", "account"},
	)

	zoneThreatsCountry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsCountryMetricName.String(),
		Help: "Threats per zone per country",
	}, []string{"zone", "account", "country", "region"},
	)

	zoneThreatsType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneThreatsTypeMetricName.String(),
		Help: "Threats per zone per type",
	}, []string{"zone", "account", "type"},
	)

	zonePageviewsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zonePageviewsTotalMetricName.String(),
		Help: "Pageviews per zone",
	}, []string{"zone", "account"},
	)

	zoneUniquesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneUniquesTotalMetricName.String(),
		Help: "Uniques per zone",
	}, []string{"zone", "account"},
	)

	zoneColocationVisits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationVisitsMetricName.String(),
		Help: "Total visits per colocation",
	}, []string{"zone", "account", "colocation", "host"},
	)

	zoneColocationEdgeResponseBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationEdgeResponseBytesMetricName.String(),
		Help: "Edge response bytes per colocation",
	}, []string{"zone", "account", "colocation", "host"},
	)

	zoneColocationRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneColocationRequestsTotalMetricName.String(),
		Help: "Total requests per colocation",
	}, []string{"zone", "account", "colocation", "host"},
	)

	zoneFirewallEventsCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneFirewallEventsCountMetricName.String(),
		Help: "Count of Firewall events",
	}, []string{"zone", "account", "action", "source", "rule", "host", "country"},
	)

	zoneHealthCheckEventsOriginCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneHealthCheckEventsOriginCountMetricName.String(),
		Help: "Number of Heath check events per region per origin",
	}, []string{"zone", "account", "health_status", "origin_ip", "region", "fqdn"},
	)

	workerRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: workerRequestsMetricName.String(),
		Help: "Number of requests sent to worker by script name",
	}, []string{"script_name", "account", "status"},
	)

	workerErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: workerErrorsMetricName.String(),
		Help: "Number of errors by script name",
	}, []string{"script_name", "account", "status"},
	)

	workerCPUTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: workerCPUTimeMetricName.String(),
		Help: "CPU time quantiles by script name",
	}, []string{"script_name", "account", "status", "quantile"},
	)

	workerDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: workerDurationMetricName.String(),
		Help: "Duration quantiles by script name (GB*s)",
	}, []string{"script_name", "account", "status", "quantile"},
	)

	poolHealthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: poolHealthStatusMetricName.String(),
		Help: "Reports the health of a pool, 1 for healthy, 0 for unhealthy.",
	},
		[]string{"zone", "account", "load_balancer_name", "pool_name"},
	)

	poolOriginHealthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: poolOriginHealthStatusMetricName.String(),
		Help: "Reports the origin health of a pool, 1 for healthy, 0 for unhealthy.",
	},
		[]string{"account", "pool_name", "origin_name", "ip"},
	)

	poolRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: poolRequestsTotalMetricName.String(),
		Help: "Requests per pool",
	},
		[]string{"zone", "account", "load_balancer_name", "pool_name", "origin_name"},
	)

	// TODO: Update this to counter vec and use counts from the query to add
	logpushFailedJobsAccount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: logpushFailedJobsAccountMetricName.String(),
		Help: "Number of failed logpush jobs on the account level",
	},
		[]string{"account", "destination", "job_id", "final"},
	)

	logpushFailedJobsZone = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: logpushFailedJobsZoneMetricName.String(),
		Help: "Number of failed logpush jobs on the zone level",
	},
		[]string{"destination", "job_id", "final"},
	)

	r2StorageTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2StorageTotalMetricName.String(),
		Help: "Total storage used by R2",
	}, []string{"account"})

	r2Storage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2StorageMetricName.String(),
		Help: "Storage used by R2",
	}, []string{"account", "bucket"})

	r2Operation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2OperationMetricName.String(),
		Help: "Number of operations performed by R2",
	}, []string{"account", "bucket", "operation"})

	tunnelInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: tunnelInfoMetricName.String(),
		Help: "Reports Cloudflare Tunnel details",
	}, []string{"account", "tunnel_id", "tunnel_name", "tunnel_type"})

	tunnelHealthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: tunnelHealthStatusMetricName.String(),
		Help: "Reports the health of a Cloudflare Tunnel, 0 for unhealthy, 1 for healthy, 2 for degraded, 3 for inactive",
	}, []string{"account", "tunnel_id"})

	tunnelConnectorInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: tunnelConnectorInfoMetricName.String(),
		Help: "Reports Cloudflare Tunnel connector details",
	}, []string{"account", "tunnel_id", "client_id", "version", "arch", "origin_ip"})

	tunnelConnectorActiveConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: tunnelConnectorActiveConnectionsMetricName.String(),
		Help: "Reports number of active connections for a Cloudflare Tunnel connector",
	}, []string{"account", "tunnel_id", "client_id"})

	metricsMap = MetricsMap{}
)

func init() {
	metricsMap[zoneRequestTotalMetricName] = zoneRequestTotal
	metricsMap[zoneRequestCachedMetricName] = zoneRequestCached
	metricsMap[zoneRequestSSLEncryptedMetricName] = zoneRequestSSLEncrypted
	metricsMap[zoneRequestContentTypeMetricName] = zoneRequestContentType
	metricsMap[zoneRequestCountryMetricName] = zoneRequestCountry
	metricsMap[zoneRequestHTTPStatusMetricName] = zoneRequestHTTPStatus
	metricsMap[zoneRequestBrowserMapMetricName] = zoneRequestBrowserMap
	metricsMap[zoneRequestOriginStatusCountryHostMetricName] = zoneRequestOriginStatusCountryHost
	metricsMap[zoneRequestStatusCountryHostMetricName] = zoneRequestStatusCountryHost
	metricsMap[zoneBandwidthTotalMetricName] = zoneBandwidthTotal
	metricsMap[zoneBandwidthCachedMetricName] = zoneBandwidthCached
	metricsMap[zoneBandwidthSSLEncryptedMetricName] = zoneBandwidthSSLEncrypted
	metricsMap[zoneBandwidthContentTypeMetricName] = zoneBandwidthContentType
	metricsMap[zoneBandwidthCountryMetricName] = zoneBandwidthCountry
	metricsMap[zoneThreatsTotalMetricName] = zoneThreatsTotal
	metricsMap[zoneThreatsCountryMetricName] = zoneThreatsCountry
	metricsMap[zoneThreatsTypeMetricName] = zoneThreatsType
	metricsMap[zonePageviewsTotalMetricName] = zonePageviewsTotal
	metricsMap[zoneUniquesTotalMetricName] = zoneUniquesTotal
	metricsMap[zoneColocationVisitsMetricName] = zoneColocationVisits
	metricsMap[zoneColocationEdgeResponseBytesMetricName] = zoneColocationEdgeResponseBytes
	metricsMap[zoneColocationRequestsTotalMetricName] = zoneColocationRequestsTotal
	metricsMap[zoneFirewallEventsCountMetricName] = zoneFirewallEventsCount
	metricsMap[zoneHealthCheckEventsOriginCountMetricName] = zoneHealthCheckEventsOriginCount
	metricsMap[workerRequestsMetricName] = workerRequests
	metricsMap[workerErrorsMetricName] = workerErrors
	metricsMap[workerCPUTimeMetricName] = workerCPUTime
	metricsMap[workerDurationMetricName] = workerDuration
	metricsMap[poolHealthStatusMetricName] = poolHealthStatus
	metricsMap[poolOriginHealthStatusMetricName] = poolOriginHealthStatus
	metricsMap[poolRequestsTotalMetricName] = poolRequestsTotal
	metricsMap[logpushFailedJobsAccountMetricName] = logpushFailedJobsAccount
	metricsMap[logpushFailedJobsZoneMetricName] = logpushFailedJobsZone
	metricsMap[r2StorageTotalMetricName] = r2StorageTotal
	metricsMap[r2StorageMetricName] = r2Storage
	metricsMap[r2OperationMetricName] = r2Operation
	metricsMap[tunnelInfoMetricName] = tunnelInfo
	metricsMap[tunnelHealthStatusMetricName] = tunnelHealthStatus
	metricsMap[tunnelConnectorInfoMetricName] = tunnelConnectorInfo
	metricsMap[tunnelConnectorActiveConnectionsMetricName] = tunnelConnectorActiveConnections
}

func buildDeniedMetricsSet(metricsDenylist []string) (MetricsMap, error) {
	out := maps.Clone(metricsMap)
	for _, metric := range metricsDenylist {
		name := MetricName(metric)
		if _, found := out[name]; !found {
			return nil, fmt.Errorf("metric %s doesn't exists", name)
		}
		delete(out, name)
	}
	return out, nil
}

func buildAllowedMetricsSet(allowList []string) (MetricsMap, error) {
	out := MetricsMap{}
	for _, metric := range allowList {
		name := MetricName(metric)
		metric, found := metricsMap[name]
		if !found {
			return nil, fmt.Errorf("metric %s doesn't exists", name)
		}
		out[name] = metric
	}
	return out, nil
}

// check if none of the `metricNames` are in `metrics` we can skip
func shouldSkip(metrics MetricsMap, metricNames ...MetricName) bool {
	for name := range metrics {
		if slices.Contains(metricNames, name) {
			return false
		}
	}
	return true
}

func fetchLoadblancerPoolsHealth(metrics MetricsMap, account cfaccounts.Account) {
	if shouldSkip(metrics, poolOriginHealthStatusMetricName) {
		return
	}

	pools := fetchLoadblancerPools(account)
	if pools == nil {
		return
	}

	for _, pool := range pools {
		if !pool.Enabled { // not enabled, no health values
			continue
		}
		if pool.Monitor == "" { // No monitor, no health values
			continue
		}
		for _, o := range pool.Origins {
			if !o.Enabled { // not enabled, no health values
				continue
			}
			healthy := 1 // Assume healthy
			if o.JSON.ExtraFields["healthy"].Raw() == "false" {
				healthy = 0 // Unhealthy
			}
			poolOriginHealthStatus.With(
				prometheus.Labels{
					"account":     account.Name,
					"pool_name":   pool.Name,
					"origin_name": o.Name,
					"ip":          o.Address,
				}).Set(float64(healthy))
		}
	}
}

func fetchWorkerAnalytics(metrics MetricsMap, account cfaccounts.Account) {
	if shouldSkip(
		metrics,
		workerRequestsMetricName,
		workerErrorsMetricName,
		workerCPUTimeMetricName,
		workerDurationMetricName,
	) {
		return
	}

	r, err := fetchWorkerTotals(account.ID)
	if err != nil {
		log.Error("failed to fetch worker analytics for account ", account.ID, ": ", err)
		return
	}

	// Replace spaces with hyphens and convert to lowercase
	accountName := strings.ToLower(strings.ReplaceAll(account.Name, " ", "-"))

	for _, a := range r.Viewer.Accounts {
		for _, w := range a.WorkersInvocationsAdaptive {
			baseLabels := prometheus.Labels{"script_name": w.Dimensions.ScriptName, "account": accountName, "status": w.Dimensions.Status}

			workerRequests.With(baseLabels).Add(float64(w.Sum.Requests))
			workerErrors.With(baseLabels).Add(float64(w.Sum.Errors))

			labeledCPUTime, _ := workerCPUTime.CurryWith(baseLabels)
			labeledCPUTime.With(prometheus.Labels{"quantile": "P50"}).Set(float64(w.Quantiles.CPUTimeP50))
			labeledCPUTime.With(prometheus.Labels{"quantile": "P75"}).Set(float64(w.Quantiles.CPUTimeP75))
			labeledCPUTime.With(prometheus.Labels{"quantile": "P99"}).Set(float64(w.Quantiles.CPUTimeP99))
			labeledCPUTime.With(prometheus.Labels{"quantile": "P999"}).Set(float64(w.Quantiles.CPUTimeP999))

			labeledDuration, _ := workerDuration.CurryWith(baseLabels)
			labeledDuration.With(prometheus.Labels{"quantile": "P50"}).Set(float64(w.Quantiles.DurationP50))
			labeledDuration.With(prometheus.Labels{"quantile": "P75"}).Set(float64(w.Quantiles.DurationP75))
			labeledDuration.With(prometheus.Labels{"quantile": "P99"}).Set(float64(w.Quantiles.DurationP99))
			labeledDuration.With(prometheus.Labels{"quantile": "P999"}).Set(float64(w.Quantiles.DurationP999))
		}
	}
}

func fetchLogpushAnalyticsForAccount(metrics MetricsMap, account cfaccounts.Account) {
	if shouldSkip(metrics, logpushFailedJobsAccountMetricName) {
		return
	}

	if viper.GetBool("free_tier") {
		return
	}

	r, err := fetchLogpushAccount(account.ID)

	if err != nil {
		log.Error("failed to fetch logpush analytics for account ", account.ID, ": ", err)
		return
	}

	for _, acc := range r.Viewer.Accounts {
		for _, LogpushHealthAdaptiveGroup := range acc.LogpushHealthAdaptiveGroups {
			logpushFailedJobsAccount.With(prometheus.Labels{"account": account.ID,
				"destination": LogpushHealthAdaptiveGroup.Dimensions.DestinationType,
				"job_id":      strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.JobID),
				"final":       strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.Final)}).Add(float64(LogpushHealthAdaptiveGroup.Count))
		}
	}
}

func fetchR2StorageForAccount(metrics MetricsMap, account cfaccounts.Account) {
	if shouldSkip(
		metrics,
		r2StorageMetricName,
		r2OperationMetricName,
		r2StorageTotalMetricName,
	) {
		return
	}

	r, err := fetchR2Account(account.ID)

	if err != nil {
		return
	}
	for _, acc := range r.Viewer.Accounts {
		var totalStorage uint64
		for _, bucket := range acc.R2StorageGroups {
			totalStorage += bucket.Max.PayloadSize
			r2Storage.With(prometheus.Labels{"account": account.Name, "bucket": bucket.Dimensions.BucketName}).Set(float64(bucket.Max.PayloadSize))
		}
		for _, operation := range acc.R2StorageOperations {
			r2Operation.With(prometheus.Labels{"account": account.Name, "bucket": operation.Dimensions.BucketName, "operation": operation.Dimensions.Action}).Set(float64(operation.Sum.Requests))
		}
		r2StorageTotal.With(prometheus.Labels{"account": account.Name}).Set(float64(totalStorage))
	}
}

func fetchLogpushAnalyticsForZone(metrics MetricsMap, zones []cfzones.Zone) {
	if shouldSkip(metrics, logpushFailedJobsZoneMetricName) {
		return
	}

	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(zones)
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchLogpushZone(zoneIDs)

	if err != nil {
		log.Error("failed to fetch logpush analytics for zones: ", err)
		return
	}

	for _, zone := range r.Viewer.Zones {
		for _, LogpushHealthAdaptiveGroup := range zone.LogpushHealthAdaptiveGroups {
			logpushFailedJobsZone.With(prometheus.Labels{
				"destination": LogpushHealthAdaptiveGroup.Dimensions.DestinationType,
				"job_id":      strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.JobID),
				"final":       strconv.Itoa(LogpushHealthAdaptiveGroup.Dimensions.Final),
			}).Add(float64(LogpushHealthAdaptiveGroup.Count))
		}
	}
}

func fetchZoneColocationAnalytics(metrics MetricsMap, zones []cfzones.Zone) {
	if shouldSkip(
		metrics,
		zoneColocationVisitsMetricName,
		zoneColocationEdgeResponseBytesMetricName,
		zoneColocationRequestsTotalMetricName,
	) {
		return
	}

	// Colocation metrics are not available in non-enterprise zones
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(zones)
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchColoTotals(zoneIDs)
	if err != nil {
		log.Error("failed to fetch colocation analytics for zones: ", err)
		return
	}
	for _, z := range r.Viewer.Zones {
		cg := z.ColoGroups
		name, account := findZoneAccountName(zones, z.ZoneTag)
		for _, c := range cg {
			label := prometheus.Labels{"zone": name, "account": account, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}
			zoneColocationVisits.With(label).Add(float64(c.Sum.Visits))
			zoneColocationEdgeResponseBytes.With(label).Add(float64(c.Sum.EdgeResponseBytes))
			zoneColocationRequestsTotal.With(label).Add(float64(c.Count))
		}
	}
}

func fetchZoneAnalytics(metrics MetricsMap, zones []cfzones.Zone) {
	if shouldSkip(
		metrics,
		zoneRequestTotalMetricName,
		zoneRequestCachedMetricName,
		zoneRequestSSLEncryptedMetricName,
		zoneRequestContentTypeMetricName,
		zoneBandwidthContentTypeMetricName,
		zoneRequestCountryMetricName,
		zoneBandwidthCountryMetricName,
		zoneThreatsCountryMetricName,
		zoneRequestHTTPStatusMetricName,
		zoneRequestBrowserMapMetricName,
		zoneBandwidthTotalMetricName,
		zoneBandwidthCachedMetricName,
		zoneBandwidthSSLEncryptedMetricName,
		zoneThreatsTotalMetricName,
		zoneThreatsTypeMetricName,
		zonePageviewsTotalMetricName,
		zoneUniquesTotalMetricName,

		zoneFirewallEventsCountMetricName,

		zoneHealthCheckEventsOriginCountMetricName,

		zoneRequestOriginStatusCountryHostMetricName,
		zoneRequestStatusCountryHostMetricName,
	) {
		return
	}

	// None of the below referenced metrics are available in the free tier
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(zones)
	if len(zoneIDs) == 0 {
		return
	}

	r, err := fetchZoneTotals(zoneIDs)
	if err != nil {
		log.Error("failed to fetch zone analytics: ", err)
		return
	}

	for _, z := range r.Viewer.Zones {
		name, account := findZoneAccountName(zones, z.ZoneTag)
		z := z

		addHTTPGroups(&z, name, account)
		addFirewallGroups(metrics, &z, name, account)
		addHealthCheckGroups(&z, name, account)
		addHTTPAdaptiveGroups(&z, name, account)
	}
}

func addHTTPGroups(z *zoneResp, name string, account string) {
	// Nothing to do.
	if len(z.HTTP1mGroups) == 0 {
		return
	}

	zt := z.HTTP1mGroups[0]

	zoneAccountLabel := prometheus.Labels{"zone": name, "account": account}
	zoneRequestTotal.With(zoneAccountLabel).Add(float64(zt.Sum.Requests))
	zoneRequestCached.With(zoneAccountLabel).Add(float64(zt.Sum.CachedRequests))
	zoneRequestSSLEncrypted.With(zoneAccountLabel).Add(float64(zt.Sum.EncryptedRequests))

	for _, ct := range zt.Sum.ContentType {
		label := prometheus.Labels{"zone": name, "account": account, "content_type": ct.EdgeResponseContentType}
		zoneRequestContentType.With(label).Add(float64(ct.Requests))
		zoneBandwidthContentType.With(label).Add(float64(ct.Bytes))
	}

	for _, country := range zt.Sum.Country {
		c := countries.ByName(country.ClientCountryName)
		region := c.Info().Region.Info().Name

		label := prometheus.Labels{"zone": name, "account": account, "country": country.ClientCountryName, "region": region}

		zoneRequestCountry.With(label).Add(float64(country.Requests))
		zoneBandwidthCountry.With(label).Add(float64(country.Bytes))
		zoneThreatsCountry.With(label).Add(float64(country.Threats))
	}

	for _, status := range zt.Sum.ResponseStatus {
		zoneRequestHTTPStatus.With(prometheus.Labels{"zone": name, "account": account, "status": strconv.Itoa(status.EdgeResponseStatus)}).Add(float64(status.Requests))
	}

	for _, browser := range zt.Sum.BrowserMap {
		zoneRequestBrowserMap.With(prometheus.Labels{"zone": name, "account": account, "family": browser.UaBrowserFamily}).Add(float64(browser.PageViews))
	}

	zoneBandwidthTotal.With(zoneAccountLabel).Add(float64(zt.Sum.Bytes))
	zoneBandwidthCached.With(zoneAccountLabel).Add(float64(zt.Sum.CachedBytes))
	zoneBandwidthSSLEncrypted.With(zoneAccountLabel).Add(float64(zt.Sum.EncryptedBytes))

	zoneThreatsTotal.With(zoneAccountLabel).Add(float64(zt.Sum.Threats))

	for _, t := range zt.Sum.ThreatPathing {
		zoneThreatsType.With(prometheus.Labels{"zone": name, "account": account, "type": t.Name}).Add(float64(t.Requests))
	}

	zonePageviewsTotal.With(zoneAccountLabel).Add(float64(zt.Sum.PageViews))

	// Uniques
	zoneUniquesTotal.With(zoneAccountLabel).Add(float64(zt.Unique.Uniques))
}

func addFirewallGroups(metrics MetricsMap, z *zoneResp, name string, account string) {
	if shouldSkip(metrics, zoneFirewallEventsCountMetricName) {
		return
	}

	// Nothing to do.
	if len(z.FirewallEventsAdaptiveGroups) == 0 {
		return
	}
	rulesMap := fetchFirewallRules(z.ZoneTag)
	for _, g := range z.FirewallEventsAdaptiveGroups {
		zoneFirewallEventsCount.With(
			prometheus.Labels{
				"zone":    name,
				"account": account,
				"action":  g.Dimensions.Action,
				"source":  g.Dimensions.Source,
				"rule":    normalizeRuleName(rulesMap[g.Dimensions.RuleID]),
				"host":    g.Dimensions.ClientRequestHTTPHost,
				"country": g.Dimensions.ClientCountryName,
			}).Add(float64(g.Count))
	}
}

func normalizeRuleName(initialText string) string {
	maxLength := 200
	nonSpaceName := strings.ReplaceAll(strings.ToLower(initialText), " ", "_")
	if len(nonSpaceName) > maxLength {
		return nonSpaceName[:maxLength]
	}
	return nonSpaceName
}

func addHealthCheckGroups(z *zoneResp, name string, account string) {
	if len(z.HealthCheckEventsAdaptiveGroups) == 0 {
		return
	}

	for _, g := range z.HealthCheckEventsAdaptiveGroups {
		zoneHealthCheckEventsOriginCount.With(
			prometheus.Labels{
				"zone":          name,
				"account":       account,
				"health_status": g.Dimensions.HealthStatus,
				"origin_ip":     g.Dimensions.OriginIP,
				"region":        g.Dimensions.Region,
				"fqdn":          g.Dimensions.Fqdn,
			}).Add(float64(g.Count))
	}
}

func addHTTPAdaptiveGroups(z *zoneResp, name string, account string) {
	for _, g := range z.HTTPRequestsAdaptiveGroups {
		zoneRequestOriginStatusCountryHost.With(
			prometheus.Labels{
				"zone":    name,
				"account": account,
				"status":  strconv.Itoa(int(g.Dimensions.OriginResponseStatus)),
				"country": g.Dimensions.ClientCountryName,
				"host":    g.Dimensions.ClientRequestHTTPHost,
			}).Add(float64(g.Count))
	}

	for _, g := range z.HTTPRequestsEdgeCountryHost {
		zoneRequestStatusCountryHost.With(
			prometheus.Labels{
				"zone":    name,
				"account": account,
				"status":  strconv.Itoa(int(g.Dimensions.EdgeResponseStatus)),
				"country": g.Dimensions.ClientCountryName,
				"host":    g.Dimensions.ClientRequestHTTPHost,
			}).Add(float64(g.Count))
	}
}

func fetchLoadBalancerAnalytics(metrics MetricsMap, zones []cfzones.Zone) {
	if shouldSkip(
		metrics,
		poolHealthStatusMetricName,
		poolRequestsTotalMetricName,
	) {
		return
	}
	// None of the below referenced metrics are available in the free tier
	if viper.GetBool("free_tier") {
		return
	}

	zoneIDs := extractZoneIDs(zones)
	if len(zoneIDs) == 0 {
		return
	}

	l, err := fetchLoadBalancerTotals(zoneIDs)
	if err != nil {
		log.Error("failed to fetch load balancer analytics: ", err)
		return
	}
	for _, lb := range l.Viewer.Zones {
		name, account := findZoneAccountName(zones, lb.ZoneTag)
		lb := lb
		addLoadBalancingRequestsAdaptive(&lb, name, account)
		addLoadBalancingRequestsAdaptiveGroups(&lb, name, account)
	}
}

func addLoadBalancingRequestsAdaptiveGroups(z *lbResp, name string, account string) {
	for _, g := range z.LoadBalancingRequestsAdaptiveGroups {
		poolRequestsTotal.With(
			prometheus.Labels{
				"zone":               name,
				"account":            account,
				"load_balancer_name": g.Dimensions.LbName,
				"pool_name":          g.Dimensions.SelectedPoolName,
				"origin_name":        g.Dimensions.SelectedOriginName,
			}).Add(float64(g.Count))
	}
}

func addLoadBalancingRequestsAdaptive(z *lbResp, name string, account string) {
	for _, g := range z.LoadBalancingRequestsAdaptive {
		for _, p := range g.Pools {
			poolHealthStatus.With(
				prometheus.Labels{
					"zone":               name,
					"account":            account,
					"load_balancer_name": g.LbName,
					"pool_name":          p.PoolName,
				}).Set(float64(p.Healthy))
		}
	}
}

func fetchZeroTrustAnalyticsForAccount(metrics MetricsMap, account cfaccounts.Account) {
	addCloudflareTunnelStatus(metrics, account)
}

func addCloudflareTunnelStatus(metrics MetricsMap, account cfaccounts.Account) {
	if shouldSkip(
		metrics,
		tunnelInfoMetricName,
		tunnelHealthStatusMetricName,
		tunnelConnectorInfoMetricName,
		tunnelConnectorActiveConnectionsMetricName,
	) {
		return
	}

	tunnels := fetchCloudflareTunnels(account)
	for _, t := range tunnels {
		tunnelInfo.With(
			prometheus.Labels{
				"account":     account.Name,
				"tunnel_id":   t.ID,
				"tunnel_name": t.Name,
				"tunnel_type": string(t.TunType),
			}).Set(float64(1))

		tunnelHealthStatus.With(
			prometheus.Labels{
				"account":   account.Name,
				"tunnel_id": t.ID,
			}).Set(float64(getCloudflareTunnelStatusValue(string(t.Status))))

		// Each client/connector can open many connections to the Cloudflare edge,
		// we opt to not expose metrics for each individual connection. We do expose
		// an informational metric for each client/connector however.
		clients := fetchCloudflareTunnelConnectors(account, t.ID)
		for _, c := range clients {
			originIP := ""
			if len(c.Conns) > 0 {
				originIP = c.Conns[0].OriginIP
			}

			tunnelConnectorInfo.With(
				prometheus.Labels{
					"account":   account.Name,
					"tunnel_id": t.ID,
					"client_id": c.ID,
					"version":   c.Version,
					"arch":      c.Arch,
					"origin_ip": originIP,
				}).Set(float64(1))

			tunnelConnectorActiveConnections.With(
				prometheus.Labels{
					"account":   account.Name,
					"tunnel_id": t.ID,
					"client_id": c.ID,
				}).Set(float64(len(c.Conns)))
		}
	}
}

// The status of the tunnel.
// Valid values are:
//   - inactive (tunnel has never been run)
//   - degraded (tunnel is active and able to serve traffic but in an unhealthy state)
//   - healthy (tunnel is active and able to serve traffic)
//   - down (tunnel can not serve traffic as it has no connections to the Cloudflare Edge).
func getCloudflareTunnelStatusValue(status string) uint8 {
	switch status {
	case "inactive":
		return 3
	case "degraded":
		return 2
	case "healthy":
		return 1
	case "down":
		return 0
	default:
		// Undefined status value returned by the API
		return 255
	}
}
