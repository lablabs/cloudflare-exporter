package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type CloudflareCollector struct {
	zoneRequestTotal    *prometheus.Desc
	zoneRequestCached   *prometheus.Desc
	zoneRequestUncached *prometheus.Desc
}

func newCloudflareCollector() *CloudflareCollector {
	return &CloudflareCollector{
		zoneRequestTotal: prometheus.NewDesc("cloudflare_zone_requests_total",
			"Number of requests for zone",
			[]string{"zone"}, nil,
		),
		zoneRequestCached: prometheus.NewDesc("cloudflare_zone_requests_cached",
			"Number of cached requests for zone",
			[]string{"zone"}, nil,
		),
		zoneRequestUncached: prometheus.NewDesc("cloudflare_zone_requests_uncached",
			"Number of uncached requests for zone",
			[]string{"zone"}, nil,
		),
	}
}

func (collector *CloudflareCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.zoneRequestTotal
	ch <- collector.zoneRequestCached
	ch <- collector.zoneRequestUncached
}

func (collector *CloudflareCollector) Collect(ch chan<- prometheus.Metric) {

	var wg sync.WaitGroup
	zones := fetchZones()
	for _, z := range zones {
		// if i == 5 {
		// 	break
		// }
		wg.Add(1)
		go func(ID string, name string) {

			zt := fetchZoneTotals(ID)

			//Write latest value for each metric in the prometheus metric channel.
			//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
			ch <- prometheus.MustNewConstMetric(collector.zoneRequestTotal, prometheus.CounterValue, float64(zt.Requests.All), name)
			ch <- prometheus.MustNewConstMetric(collector.zoneRequestCached, prometheus.CounterValue, float64(zt.Requests.Cached), name)
			ch <- prometheus.MustNewConstMetric(collector.zoneRequestUncached, prometheus.CounterValue, float64(zt.Requests.Uncached), name)
			defer wg.Done()

		}(z.ID, z.Name)
	}
	wg.Wait()

}
