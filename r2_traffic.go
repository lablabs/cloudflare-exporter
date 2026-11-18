package main

import (
	"context"
	"fmt"
	"time"

	cfaccounts "github.com/cloudflare/cloudflare-go/v4/accounts"
	"github.com/machinebox/graphql"
	"github.com/prometheus/client_golang/prometheus"
)

const r2TrafficWindow = 5 * time.Minute

var (
	r2Upload = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2UploadMetricName.String(),
		Help: "Five-minute average uploaded bytes per second; excludes transfers smaller than 100 KiB",
	}, []string{"account", "bucket"})
	r2Download = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2DownloadMetricName.String(),
		Help: "Five-minute average downloaded bytes per second; excludes transfers smaller than 100 KiB",
	}, []string{"account", "bucket"})
	r2ErrorRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2ErrorRatioMetricName.String(),
		Help: "Fraction of recorded requests that failed in the five-minute window; zero for idle buckets; 429 is included in 4xx",
	}, []string{"account", "bucket", "status_class"})
	r2TrafficUpdated = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r2TrafficUpdatedMetricName.String(),
		Help: "Unix timestamp of the latest successful traffic collection; zero when the latest attempt failed",
	}, []string{"account"})
)

type r2TrafficAccount struct {
	Bandwidth []struct {
		Dimensions struct{ BucketName string }
		Sum        struct{ BytesUpload, BytesDownload uint64 }
	}
	Operations []struct {
		Dimensions struct {
			BucketName, ActionStatus string
			ResponseStatusCode       int
		}
		Sum struct{ Requests uint64 }
	}
}

type r2BucketTraffic struct {
	upload, download, requests uint64
	errors                     map[string]uint64
}

func aggregateR2Traffic(account r2TrafficAccount, buckets []string) (map[string]*r2BucketTraffic, error) {
	traffic := make(map[string]*r2BucketTraffic)
	bucket := func(name string) *r2BucketTraffic {
		if traffic[name] == nil {
			traffic[name] = &r2BucketTraffic{errors: map[string]uint64{"all": 0, "4xx": 0, "5xx": 0, "429": 0, "other": 0}}
		}
		return traffic[name]
	}
	for _, name := range buckets {
		bucket(name)
	}
	for _, row := range account.Bandwidth {
		b := bucket(row.Dimensions.BucketName)
		b.upload += row.Sum.BytesUpload
		b.download += row.Sum.BytesDownload
	}
	for _, row := range account.Operations {
		status := row.Dimensions.ActionStatus
		if status != "success" && status != "userError" && status != "internalError" {
			return nil, fmt.Errorf("unknown R2 action status %q", status)
		}
		b := bucket(row.Dimensions.BucketName)
		b.requests += row.Sum.Requests
		code := row.Dimensions.ResponseStatusCode
		if code >= 400 || status != "success" {
			b.errors["all"] += row.Sum.Requests
			category := "other"
			if code >= 400 && code < 500 {
				category = "4xx"
			}
			if code >= 500 && code < 600 {
				category = "5xx"
			}
			b.errors[category] += row.Sum.Requests
			if code == 429 {
				b.errors["429"] += row.Sum.Requests
			}
		}
	}
	return traffic, nil
}

func clearR2Traffic(account string) {
	labels := prometheus.Labels{"account": account}
	r2Upload.DeletePartialMatch(labels)
	r2Download.DeletePartialMatch(labels)
	r2ErrorRatio.DeletePartialMatch(labels)
	r2TrafficUpdated.WithLabelValues(account).Set(0)
}

func fetchR2Traffic(accountID string) (r2TrafficAccount, error) {
	request := graphql.NewRequest(`query($accountID: String!, $limit: Int!, $mintime: Time!, $maxtime: Time!) {
        viewer { accounts(filter: {accountTag: $accountID}) {
            bandwidth: r2BandwidthUsageAdaptiveGroups(limit: $limit, filter: {
                datetime_geq: $mintime, datetime_lt: $maxtime
            }) { dimensions { bucketName } sum { bytesUpload bytesDownload } }
            operations: r2OperationsAdaptiveGroups(limit: $limit, filter: {
                datetime_geq: $mintime, datetime_lt: $maxtime
            }) { dimensions { bucketName actionStatus responseStatusCode } sum { requests } }
        } }
    }`)
	end, _ := GetTimeRange()
	request.Var("accountID", accountID)
	request.Var("limit", gqlQueryLimit)
	request.Var("mintime", end.Add(-r2TrafficWindow))
	request.Var("maxtime", end)
	var response struct {
		Viewer struct{ Accounts []r2TrafficAccount }
	}
	ctx, cancel := context.WithTimeout(context.Background(), cftimeout)
	defer cancel()
	gql.Mu.RLock()
	defer gql.Mu.RUnlock()
	if err := gql.Client.Run(ctx, request, &response); err != nil {
		return r2TrafficAccount{}, fmt.Errorf("query R2 traffic: %w", err)
	}
	if len(response.Viewer.Accounts) != 1 {
		return r2TrafficAccount{}, fmt.Errorf("expected one R2 analytics account, got %d", len(response.Viewer.Accounts))
	}
	data := response.Viewer.Accounts[0]
	if len(data.Bandwidth) >= gqlQueryLimit || len(data.Operations) >= gqlQueryLimit {
		return r2TrafficAccount{}, fmt.Errorf("R2 traffic query reached group limit %d", gqlQueryLimit)
	}
	return data, nil
}

func fetchR2TrafficForAccount(account cfaccounts.Account, buckets []string) {
	data, err := fetchR2Traffic(account.ID)
	var traffic map[string]*r2BucketTraffic
	if err == nil {
		traffic, err = aggregateR2Traffic(data, buckets)
	}
	if err != nil {
		clearR2Traffic(account.Name)
		log.Errorf("failed to fetch R2 traffic for account %s: %v", account.ID, err)
		return
	}
	clearR2Traffic(account.Name)
	for name, b := range traffic {
		r2Upload.WithLabelValues(account.Name, name).Set(float64(b.upload) / r2TrafficWindow.Seconds())
		r2Download.WithLabelValues(account.Name, name).Set(float64(b.download) / r2TrafficWindow.Seconds())
		for category, count := range b.errors {
			ratio := float64(0)
			if b.requests > 0 {
				ratio = float64(count) / float64(b.requests)
			}
			r2ErrorRatio.WithLabelValues(account.Name, name, category).Set(ratio)
		}
	}
	r2TrafficUpdated.WithLabelValues(account.Name).Set(float64(time.Now().Unix()))
}
