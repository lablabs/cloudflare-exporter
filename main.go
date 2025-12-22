package main

import (
	"net/http"
	_ "net/http/pprof" // #nosec G108 - pprof is controlled via enable_pprof flag
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/nelkinda/health-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cf "github.com/cloudflare/cloudflare-go/v4"
	cfoption "github.com/cloudflare/cloudflare-go/v4/option"
	cfzones "github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/sirupsen/logrus"
)

var (
	cfclient  *cf.Client
	cftimeout time.Duration
	gql       *GraphQL
	log       = logrus.New()
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
// 	cfgMetricsDenylist = ""
// )

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

func filterZones(all []cfzones.Zone, target []string) []cfzones.Zone {
	var filtered []cfzones.Zone

	if (len(target)) == 0 {
		return all
	}

	for _, tz := range target {
		for _, z := range all {
			if tz == z.ID {
				filtered = append(filtered, z)
				log.Debug("Filtering zone: ", z.ID, " ", z.Name)
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

func filterExcludedZones(all []cfzones.Zone, exclude []string) []cfzones.Zone {
	var filtered []cfzones.Zone

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
	accounts := fetchAccounts()

	for _, a := range accounts {
		wg.Add(1)
		go fetchWorkerAnalytics(a, &wg)

		wg.Add(1)
		go fetchLogpushAnalyticsForAccount(a, &wg)

		wg.Add(1)
		go fetchR2StorageForAccount(a, &wg)

		wg.Add(1)
		go fetchLoadblancerPoolsHealth(a, &wg)

		wg.Add(1)
		go fetchZeroTrustAnalyticsForAccount(a, &wg)
	}

	zones := fetchZones(accounts)
	tzones := getTargetZones()
	fzones := filterZones(zones, tzones)
	ezones := getExcludedZones()
	filteredZones := filterExcludedZones(fzones, ezones)
	if !viper.GetBool("free_tier") {
		filteredZones = filterNonFreePlanZones(filteredZones)
	}

	zoneCount := len(filteredZones)
	if zoneCount > 0 && zoneCount <= cfgraphqlreqlimit {
		wg.Add(1)
		go fetchZoneAnalytics(filteredZones, &wg)

		wg.Add(1)
		go fetchZoneColocationAnalytics(filteredZones, &wg)

		wg.Add(1)
		go fetchLoadBalancerAnalytics(filteredZones, &wg)

		wg.Add(1)
		go fetchLogpushAnalyticsForZone(filteredZones, &wg)

		wg.Add(1)
		go fetchZoneASNAnalytics(filteredZones, &wg)
	} else if zoneCount > cfgraphqlreqlimit {
		for s := 0; s < zoneCount; s += cfgraphqlreqlimit {
			e := s + cfgraphqlreqlimit
			if e > zoneCount {
				e = zoneCount
			}
			wg.Add(1)
			go fetchZoneAnalytics(filteredZones[s:e], &wg)

			wg.Add(1)
			go fetchZoneColocationAnalytics(filteredZones[s:e], &wg)

			wg.Add(1)
			go fetchLoadBalancerAnalytics(filteredZones[s:e], &wg)

			wg.Add(1)
			go fetchLogpushAnalyticsForZone(filteredZones[s:e], &wg)

			wg.Add(1)
			go fetchZoneASNAnalytics(filteredZones[s:e], &wg)
		}
	}

	wg.Wait()
}

func runExporter() {
	cfgMetricsPath := viper.GetString("metrics_path")

	// Handle pprof configuration
	if !viper.GetBool("enable_pprof") {
		// Remove pprof handlers from default mux if disabled
		http.DefaultServeMux = http.NewServeMux()
		log.Info("pprof disabled")
	} else {
		log.Warn("pprof enabled - profiling endpoints available at /debug/pprof/")
	}

	metricsDenylist := []string{}
	if len(viper.GetString("metrics_denylist")) > 0 {
		metricsDenylist = strings.Split(viper.GetString("metrics_denylist"), ",")
	}
	metricsSet, err := buildFilteredMetricsSet(metricsDenylist)
	if err != nil {
		log.Fatalf("Error building metrics set: %v", err)
	}
	log.Debugf("Metrics set: %v", metricsSet)
	mustRegisterMetrics(metricsSet)

	scrapeInterval := time.Duration(viper.GetInt("scrape_interval")) * time.Second
	log.Info("Scrape interval set to ", scrapeInterval)

	go func() {
		for ; true; <-time.NewTicker(scrapeInterval).C {
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
	cmd := &cobra.Command{
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

	flags.String("metrics_path", "/metrics", "path for metrics, default /metrics")
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

	flags.Int("scrape_delay", 300, "scrape delay in seconds, defaults to 300")
	viper.BindEnv("scrape_delay")
	viper.SetDefault("scrape_delay", 300)

	flags.Int("scrape_interval", 60, "scrape interval in seconds, defaults to 60")
	viper.BindEnv("scrape_interval")
	viper.SetDefault("scrape_interval", 60)

	flags.Bool("free_tier", false, "scrape only metrics included in free plan")
	viper.BindEnv("free_tier")
	viper.SetDefault("free_tier", false)

	flags.Duration("cf_timeout", 10*time.Second, "cloudflare request timeout, default 10 seconds")
	viper.BindEnv("cf_timeout")
	viper.SetDefault("cf_timeout", 10*time.Second)

	flags.String("metrics_denylist", "", "metrics to not expose, comma delimited list")
	viper.BindEnv("metrics_denylist")
	viper.SetDefault("metrics_denylist", "")

	flags.String("log_level", "info", "log level")
	viper.BindEnv("log_level")
	viper.SetDefault("log_level", "info")

	flags.Bool("enable_pprof", false, "enable pprof profiling endpoints at /debug/pprof/")
	viper.BindEnv("enable_pprof")
	viper.SetDefault("enable_pprof", false)

	viper.BindPFlags(flags)

	logLevel := viper.GetString("log_level")
	switch logLevel {
	case "debug":
		log.Level = logrus.DebugLevel
		log.SetReportCaller(true)
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	default:
		log.Level = logrus.InfoLevel
	}

	log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			funcPath := strings.Split(f.File, "/")
			file := funcPath[len(funcPath)-1]
			return "file:" + file, " func:" + f.Function
		},
	})

	cftimeout = viper.GetDuration("cf_timeout")

	if len(viper.GetString("cf_api_token")) > 0 {
		cfclient = cf.NewClient(
			cfoption.WithAPIToken(viper.GetString("cf_api_token")),
			cfoption.WithRequestTimeout(cftimeout),
		)
		middlewares := NewHeaderMiddleware("Authorization", "Bearer "+viper.GetString("cf_api_token"), http.DefaultTransport)
		gqlHTTPClient := &http.Client{
			Timeout:   cftimeout,
			Transport: middlewares,
		}
		gql = NewGraphQLClient(gqlHTTPClient)
	} else if len(viper.GetString("cf_api_email")) > 0 && len(viper.GetString("cf_api_key")) > 0 {
		cfclient = cf.NewClient(
			cfoption.WithAPIKey(viper.GetString("cf_api_key")),
			cfoption.WithAPIEmail(viper.GetString("cf_api_email")),
			cfoption.WithRequestTimeout(cftimeout),
		)
		authEmailHeader := NewHeaderMiddleware("X-AUTH-EMAIL", viper.GetString("cf_api_email"), http.DefaultTransport)
		middlewares := NewHeaderMiddleware("X-AUTH-KEY", viper.GetString("cf_api_key"), authEmailHeader)
		gqlHTTPClient := &http.Client{
			Timeout:   cftimeout,
			Transport: middlewares,
		}
		gql = NewGraphQLClient(gqlHTTPClient)
	} else {
		log.Fatal("Please provide CF_API_KEY+CF_API_EMAIL or CF_API_TOKEN")
	}

	cmd.Execute()
}
