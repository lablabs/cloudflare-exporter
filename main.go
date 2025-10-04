package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/nelkinda/health-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
)

// var (
// 	cfgListen          = ":8080"
// 	cfgCfAPIKey        = ""
// 	cfgCfAPIEmail      = ""
// 	cfgCfAPIToken      = ""
// 	cfgMetricsPath     = "/metrics"
// 	cfgZones           = ""
// 	cfgExcludeZones    = ""
// 	cfgScrapeDelay     = 300
// 	cfgFreeTier        = false
// 	cfgBatchSize       = 10
// 	cfgMetricsDenylist = ""
// )

var (
	cloudflareAPI *cloudflare.API
)

func getTargetZones() []string {
	var zoneIDs []string

	if len(viper.GetString("cf_zones")) > 0 {
		zoneIDs = strings.Split(viper.GetString("cf_zones"), ",")
	}
	return zoneIDs
}

func getExcludedZones() []string {
	var zoneIDs []string

	if len(viper.GetString("cf_exclude_zones")) > 0 {
		zoneIDs = strings.Split(viper.GetString("cf_exclude_zones"), ",")
	}
	return zoneIDs
}

func filterZones(all []cloudflare.Zone, target []string) []cloudflare.Zone {
	var filtered []cloudflare.Zone

	if (len(target)) == 0 {
		return all
	}

	for _, tz := range target {
		for _, z := range all {
			if tz == z.ID {
				filtered = append(filtered, z)
				log.Info("Filtering zone: ", z.ID, " ", z.Name)
			}
		}
	}

	return filtered
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func filterExcludedZones(all []cloudflare.Zone, exclude []string) []cloudflare.Zone {
	var filtered []cloudflare.Zone

	if (len(exclude)) == 0 {
		return all
	}

	for _, z := range all {
		if contains(exclude, z.ID) {
			log.Info("Exclude zone: ", z.ID, " ", z.Name)
		} else {
			filtered = append(filtered, z)
		}
	}

	return filtered
}

func fetchMetrics() {
	var wg sync.WaitGroup
	zones := fetchZones()
	accounts := fetchAccounts()
	filteredZones := filterExcludedZones(filterZones(zones, getTargetZones()), getExcludedZones())

	for _, a := range accounts {
		go fetchWorkerAnalytics(a, &wg)
		go fetchLogpushAnalyticsForAccount(a, &wg)
		go fetchR2StorageForAccount(a, &wg)
	}

	// Make requests in groups of cfgBatchSize to avoid rate limit
	// 10 is the maximum amount of zones you can request at once
	for len(filteredZones) > 0 {
		sliceLength := viper.GetInt("cf_batch_size")
		if len(filteredZones) < viper.GetInt("cf_batch_size") {
			sliceLength = len(filteredZones)
		}

		targetZones := filteredZones[:sliceLength]
		filteredZones = filteredZones[len(targetZones):]

		go fetchZoneAnalytics(targetZones, &wg)
		go fetchZoneColocationAnalytics(targetZones, &wg)
		go fetchLoadBalancerAnalytics(targetZones, &wg)
		go fetchLogpushAnalyticsForZone(targetZones, &wg)
	}

	wg.Wait()
}

func runExporter() {
	cfgMetricsPath := viper.GetString("metrics_path")

	if !(len(viper.GetString("cf_api_token")) > 0 || (len(viper.GetString("cf_api_email")) > 0 && len(viper.GetString("cf_api_key")) > 0)) {
		log.Fatal("Please provide CF_API_KEY+CF_API_EMAIL or CF_API_TOKEN")
	}
	if viper.GetInt("cf_batch_size") < 1 || viper.GetInt("cf_batch_size") > 10 {
		log.Fatal("CF_BATCH_SIZE must be between 1 and 10")
	}
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	logLevel, err := log.ParseLevel(viper.GetString("log_level"))
	if err != nil {
		log.Fatalf("Invalid log level: %s", viper.GetString("log_level"))
	}
	log.SetLevel(logLevel)

	if len(viper.GetString("cf_api_token")) > 0 {
		cloudflareAPI, err = cloudflare.NewWithAPIToken(viper.GetString("cf_api_token"))

	} else {
		cloudflareAPI, err = cloudflare.New(viper.GetString("cf_api_key"), viper.GetString("cf_api_email"))
	}
	if err != nil {
		log.Fatalf("Error creating Cloudflare API client: %s", err)
	}

	if len(viper.GetString("cf_api_token")) > 0 {
		status, err := cloudflareAPI.VerifyAPIToken(context.Background())
		if err != nil {
			log.Fatalf("Error creating Cloudflare API client: %s", err)
		}
		log.Debugf("API Token status: %s", status.Status)
	}

	var metricsDenylist []string
	if len(viper.GetString("metrics_denylist")) > 0 {
		metricsDenylist = strings.Split(viper.GetString("metrics_denylist"), ",")
	}
	metricsSet, err := buildFilteredMetricsSet(metricsDenylist)
	if err != nil {
		log.Fatalf("Error building metrics set: %s", err)
	}
	log.Debugf("Metrics set: %v", metricsSet)
	mustRegisterMetrics(metricsSet)

	go func() {
		for ; true; <-time.NewTicker(60 * time.Second).C {
			go fetchMetrics()
		}
	}()

	// This section will start the HTTP server and expose
	// any metrics on the /metrics endpoint.
	if !strings.HasPrefix(viper.GetString("metrics_path"), "/") {
		cfgMetricsPath = "/" + viper.GetString("metrics_path")
	}

	http.Handle(cfgMetricsPath, promhttp.Handler())
	h := health.New(health.Health{})
	http.HandleFunc("/health", h.Handler)

	log.Info("Beginning to serve metrics on ", viper.GetString("listen"), cfgMetricsPath)

	server := &http.Server{
		Addr:              viper.GetString("listen"),
		ReadHeaderTimeout: 3 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func main() {
	var cmd = &cobra.Command{
		Use:   "cloudflare_exporter",
		Short: "Prometheus exporter exposing Cloudflare Analytics dashboard data on a per-zone basis, as well as Worker metrics",
		Run: func(_ *cobra.Command, _ []string) {
			runExporter()
		},
	}

	viper.AutomaticEnv()

	flags := cmd.Flags()

	flags.String("listen", ":8080", "listen on addr:port (default :8080), omit addr to listen on all interfaces")
	viper.BindEnv("listen")
	viper.SetDefault("listen", ":8080")

	flags.String("metrics_path", "/metrics", "path for metrics")
	viper.BindEnv("metrics_path")
	viper.SetDefault("metrics_path", "/metrics")

	flags.String("cf_api_key", "", "cloudflare api key, required with api_email flag")
	viper.BindEnv("cf_api_key")

	flags.String("cf_api_email", "", "cloudflare api email, required with api_key flag")
	viper.BindEnv("cf_api_email")

	flags.String("cf_api_token", "", "cloudflare api token (preferred)")
	viper.BindEnv("cf_api_token")

	flags.String("cf_zones", "", "cloudflare zones to export, comma delimited list of zone ids")
	viper.BindEnv("cf_zones")
	viper.SetDefault("cf_zones", "")

	flags.String("cf_exclude_zones", "", "cloudflare zones to exclude, comma delimited list of zone ids")
	viper.BindEnv("cf_exclude_zones")
	viper.SetDefault("cf_exclude_zones", "")

	flags.Int("scrape_delay", 300, "scrape delay in seconds0")
	viper.BindEnv("scrape_delay")
	viper.SetDefault("scrape_delay", 300)

	flags.Int("cf_batch_size", 10, "cloudflare zones batch size (1-10)")
	viper.BindEnv("cf_batch_size")
	viper.SetDefault("cf_batch_size", 10)

	flags.Bool("free_tier", false, "scrape only metrics included in free plan")
	viper.BindEnv("free_tier")
	viper.SetDefault("free_tier", false)

	flags.String("metrics_denylist", "", "metrics to not expose, comma delimited list")
	viper.BindEnv("metrics_denylist")
	viper.SetDefault("metrics_denylist", "")

	flags.String("log_level", "info", "log level")
	viper.BindEnv("log_level")
	viper.SetDefault("log_level", "info")

	viper.BindPFlags(flags)
	cmd.Execute()
}
