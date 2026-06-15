package main

import (
	"context"
	"errors"
	"net/http"
	_ "net/http/pprof" // #nosec G108 - pprof is controlled via enable_pprof flag
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
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

func getTargetAccounts() []string {
	var accountIDs []string

	if len(viper.GetString("cf_accounts")) > 0 {
		accountIDs = strings.Split(viper.GetString("cf_accounts"), ",")
	}
	return accountIDs
}

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

func fetchMetrics(ctx context.Context) {
	var wg sync.WaitGroup
	targetAccounts := getTargetAccounts()
	accounts := fetchAccounts(ctx, targetAccounts)

	for _, a := range accounts {
		wg.Go(func() { fetchWorkerAnalytics(ctx, a) })
		wg.Go(func() { fetchLogpushAnalyticsForAccount(ctx, a) })
		wg.Go(func() { fetchR2StorageForAccount(ctx, a) })
		wg.Go(func() { fetchLoadblancerPoolsHealth(ctx, a) })
		wg.Go(func() { fetchZeroTrustAnalyticsForAccount(ctx, a) })
		wg.Go(func() { fetchAccountHTTPDataTransferAnalytics(ctx, a) })
	}

	zones := fetchZones(ctx, accounts)
	tzones := getTargetZones()
	fzones := filterZones(zones, tzones)
	ezones := getExcludedZones()
	filteredZones := filterExcludedZones(fzones, ezones)
	if !viper.GetBool("free_tier") {
		filteredZones = filterNonFreePlanZones(filteredZones)
	}

	zoneCount := len(filteredZones)
	if zoneCount > 0 && zoneCount <= cfgraphqlreqlimit {
		fetchZoneMetrics(ctx, &wg, filteredZones)
	} else if zoneCount > cfgraphqlreqlimit {
		for s := 0; s < zoneCount; s += cfgraphqlreqlimit {
			e := s + cfgraphqlreqlimit
			if e > zoneCount {
				e = zoneCount
			}
			fetchZoneMetrics(ctx, &wg, filteredZones[s:e])
		}
	}

	wg.Wait()
}

// fetchZoneMetrics fans out every zone-scoped collector for a single batch of
// zones, each on its own goroutine tracked by wg.
func fetchZoneMetrics(ctx context.Context, wg *sync.WaitGroup, zones []cfzones.Zone) {
	wg.Go(func() { fetchZoneAnalytics(ctx, zones) })
	wg.Go(func() { fetchZoneColocationAnalytics(ctx, zones) })
	wg.Go(func() { fetchLoadBalancerAnalytics(ctx, zones) })
	wg.Go(func() { fetchLogpushAnalyticsForZone(ctx, zones) })
	wg.Go(func() { fetchZoneASNAnalytics(ctx, zones) })
	wg.Go(func() { fetchEdgeErrorsByPathAnalytics(ctx, zones) })
}

func runExporter() error {
	// The exporter owns the whole process lifecycle: derive a context that is
	// cancelled on SIGINT/SIGTERM so we can drain connections on shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
		ticker := time.NewTicker(scrapeInterval)
		defer ticker.Stop()
		for {
			// Run synchronously: the ticker coalesces ticks while a scrape is
			// in flight, so a slow scrape can't pile up overlapping runs, and
			// ctx cancellation unwinds the in-flight scrape before we return.
			fetchMetrics(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
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

	// Serve in the background so we can block on either a server error or a
	// shutdown signal. http.ErrServerClosed is the expected, clean return
	// value once Shutdown has been called.
	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("Shutdown signal received, draining connections...")
		// Fresh context: ctx is already cancelled. Bound the drain so a stuck
		// connection cannot hang shutdown forever.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		log.Info("Shutdown complete")
		return nil
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "cloudflare_exporter",
		Short: "Prometheus exporter exposing Cloudflare Analytics dashboard data on a per-zone basis, as well as Worker metrics",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runExporter()
		},
		// runExporter errors are reported via log.Fatal below; don't let cobra
		// also print the error or the usage text on a runtime failure.
		SilenceErrors: true,
		SilenceUsage:  true,
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

	flags.String("cf_accounts", "", "cloudflare accounts to monitor, comma delimited list of account ids (required for account-scoped API tokens)")
	viper.BindEnv("cf_accounts")
	viper.SetDefault("cf_accounts", "")

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

	flags.Bool("enable_edge_errors_by_path", false, "enable edge errors by path metric (high cardinality)")
	viper.BindEnv("enable_edge_errors_by_path")
	viper.SetDefault("enable_edge_errors_by_path", false)

	flags.Bool("enable_account_usage_metrics", false, "enable account-level current calendar month HTTP data transfer metrics")
	viper.BindEnv("enable_account_usage_metrics")
	viper.SetDefault("enable_account_usage_metrics", false)

	flags.String("account_usage_request_source", "eyeball", "Cloudflare requestSource value for account usage metrics")
	viper.BindEnv("account_usage_request_source")
	viper.SetDefault("account_usage_request_source", "eyeball")

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

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
