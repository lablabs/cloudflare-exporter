# CloudFlare Prometheus exporter

[<img src="ll-logo.png">](https://lablabs.io/)

We help companies build, run, deploy and scale software and infrastructure by embracing the right technologies and principles. Check out our website at <https://lablabs.io/>

---

## Description

Prometheus exporter exposing Cloudflare Analytics dashboard data on a per-zone basis, as well as Worker metrics.
The exporter is also able to scrape Zone metrics by Colocations (<https://www.cloudflare.com/network/>).

## Grafana Dashboard

![Dashboard](https://i.ibb.co/HDsqDF1/cf-exporter.png)

Our public dashboard is available at <https://grafana.com/grafana/dashboards/13133>

## Authentication

Authentication towards the Cloudflare API can be done in two ways:

### API token

The preferred way of authenticating is with an API token, for which the scope can be configured at the Cloudflare
dashboard.

Required authentication scopes:

- `Zone/Analytics:Read` is required for zone-level metrics
- `Account/Account Analytics:Read` is required for Worker metrics
- `Account/Account Settings:Read` is required for Worker metrics (for listing accessible accounts, scraping all available
  Workers included in authentication scope)
- `Zone/Firewall Services:Read` is required to fetch zone rule name for `cloudflare_zone_firewall_events_count` metric
- `Account/Account Rulesets:Read` is required to fetch account rule name for `cloudflare_zone_firewall_events_count` metric
- `Account:Load Balancing: Monitors and Pools:Read` is required to fetch pools origin health status `cloudflare_pool_origin_health_status` metric
- `Cloudflare Tunnel Read` is required to fetch Cloudflare Tunnel (Cloudflare Zero Trust) metrics

To authenticate this way, only set `CF_API_TOKEN` (omit `CF_API_EMAIL` and `CF_API_KEY`)

[Shortcut to create the API token](https://dash.cloudflare.com/profile/api-tokens?permissionGroupKeys=%5B%7B%22key%22%3A%22account_analytics%22%2C%22type%22%3A%22read%22%7D%2C%7B%22key%22%3A%22account_settings%22%2C%22type%22%3A%22read%22%7D%2C%7B%22key%22%3A%22analytics%22%2C%22type%22%3A%22read%22%7D%2C%7B%22key%22%3A%22firewall_services%22%2C%22type%22%3A%22read%22%7D%5D&name=Cloudflare+Exporter&accountId=*&zoneId=all)

### User email + API key

To authenticate with user email + API key, use the `Global API Key` from the Cloudflare dashboard.
Beware that this key authenticates with write access to every Cloudflare resource.

To authenticate this way, set both `CF_API_KEY` and `CF_API_EMAIL`.

## Configuration

The exporter can be configured using env variables or command flags.

| **KEY** | **description** |
|-|-|
| `CF_API_EMAIL` |  user email (see <https://support.cloudflare.com/hc/en-us/articles/200167836-Managing-API-Tokens-and-Keys>) |
| `CF_API_KEY` |  API key associated with email (`CF_API_EMAIL` is required if this is set)|
| `CF_API_TOKEN` |  API authentication token (recommended before API key + email. Version 0.0.5+. see <https://developers.cloudflare.com/analytics/graphql-api/getting-started/authentication/api-token-auth>) |
| `CF_ZONES` |  (Optional) cloudflare zones to export, comma delimited list of zone ids. If not set, all zones from account are exported |
| `CF_EXCLUDE_ZONES` |  (Optional) cloudflare zones to exclude, comma delimited list of zone ids. If not set, no zones from account are excluded |
| `CF_TIMEOUT` | Set cloudflare request timeout. Default 10 seconds |
| `FREE_TIER` | (Optional) scrape only metrics included in free plan. Accepts `true` or `false`, default `false`. |
| `LISTEN` |  listen on addr:port (default `:8080`), omit addr to listen on all interfaces |
| `METRICS_PATH` |  path for metrics, default `/metrics` |
| `SCRAPE_DELAY` | scrape delay in seconds, default `300` |
| `SCRAPE_INTERVAL` | scrape interval in seconds (will query cloudflare every SCRAPE_INTERVAL seconds), default `60` |
| `METRICS_DENYLIST` | (Optional) cloudflare-exporter metrics to not export, comma delimited list of cloudflare-exporter metrics. If not set, all metrics are exported. Cannot not be used with METRICS_ALLOWLIST. |
| `METRICS_ALLOWLIST` | (Optional) cloudflare-exporter metrics to export *only*, comma delimited list of cloudflare-exporter metrics. If not set, all metrics are exported Cannot not be used with METRICS_DENYLIST. |
| `ENABLE_PPROF` | (Optional) enable pprof profiling endpoints at `/debug/pprof/`. Accepts `true` or `false`, default `false`. **Warning**: Only enable in development/debugging environments |
| `ZONE_<NAME>` |  `DEPRECATED since 0.0.5` (optional) Zone ID. Add zones you want to scrape by adding env vars in this format. You can find the zone ids in Cloudflare dashboards. |
| `LOG_LEVEL` | Set loglevel. Options are error, warn, info, debug. default `error` |

Corresponding flags:

```
  -cf_api_email="": cloudflare api email, works with api_key flag
  -cf_api_key="": cloudflare api key, works with api_email flag
  -cf_api_token="": cloudflare api token (version 0.0.5+, preferred)
  -cf_zones="": cloudflare zones to export, comma delimited list
  -cf_exclude_zones="": cloudflare zones to exclude, comma delimited list
  -cf_timeout="10s": cloudflare request timeout, default 10 seconds
  -free_tier=false: scrape only metrics included in free plan, default false
  -listen=":8080": listen on addr:port ( default :8080), omit addr to listen on all interfaces
  -metrics_path="/metrics": path for metrics, default /metrics
  -scrape_delay=300: scrape delay in seconds, defaults to 300
  -scrape_interval=60: scrape interval in seconds, defaults to 60
  -metrics_denylist="": cloudflare-exporter metrics to not export, comma delimited list
  -metrics_allowlist="": cloudflare-exporter metrics to export only, comma delimited list
  -enable_pprof=false: enable pprof profiling endpoints at /debug/pprof/
  -log_level="error": log level(error,warn,info,debug)
```

Note: `ZONE_<name>` configuration is not supported as flag.

## List of available metrics

```
# HELP cloudflare_worker_cpu_time CPU time quantiles by script name
# HELP cloudflare_worker_duration Duration quantiles by script name (GB*s)
# HELP cloudflare_worker_errors_count Number of errors by script name
# HELP cloudflare_worker_requests_count Number of requests sent to worker by script name
# HELP cloudflare_zone_bandwidth_cached Cached bandwidth per zone in bytes
# HELP cloudflare_zone_bandwidth_content_type Bandwidth per zone per content type
# HELP cloudflare_zone_bandwidth_country Bandwidth per country per zone
# HELP cloudflare_zone_bandwidth_ssl_encrypted Encrypted bandwidth per zone in bytes
# HELP cloudflare_zone_bandwidth_total Total bandwidth per zone in bytes
# HELP cloudflare_zone_colocation_edge_response_bytes Edge response bytes per colocation
# HELP cloudflare_zone_colocation_visits Total visits per colocation
# HELP cloudflare_zone_colocation_requests_total Total requests per colocation
# HELP cloudflare_zone_pageviews_total Pageviews per zone
# HELP cloudflare_zone_requests_cached Number of cached requests for zone
# HELP cloudflare_zone_requests_content_type Number of request for zone per content type
# HELP cloudflare_zone_requests_country Number of request for zone per country
# HELP cloudflare_zone_requests_origin_status_country_host Count of not cached requests for zone per origin HTTP status per country per host
# HELP cloudflare_zone_requests_ssl_encrypted Number of encrypted requests for zone
# HELP cloudflare_zone_requests_status Number of request for zone per HTTP status
# HELP cloudflare_zone_requests_status_country_host Count of requests for zone per edge HTTP status per country per host
# HELP cloudflare_zone_requests_browser_map_page_views_count Number of successful requests for HTML pages per zone
# HELP cloudflare_zone_requests_total Number of requests for zone
# HELP cloudflare_zone_threats_country Threats per zone per country
# HELP cloudflare_zone_threats_total Threats per zone
# HELP cloudflare_zone_uniques_total Uniques per zone
# HELP cloudflare_zone_pool_health_status Reports the health of a pool, 1 for healthy, 0 for unhealthy
# HELP cloudflare_zone_pool_requests_total Requests per pool
# HELP cloudflare_logpush_failed_jobs_account_count Number of failed logpush jobs on the account level
# HELP cloudflare_logpush_failed_jobs_zone_count Number of failed logpush jobs on the zone level
# HELP cloudflare_r2_operation_count Number of operations performed by R2
# HELP cloudflare_r2_storage_bytes Storage used by R2
# HELP cloudflare_r2_storage_total_bytes Total storage used by R2
```

## Helm chart repository

To deploy the exporter into Kubernetes, we recommend using our manager Helm repository:

```
helm repo add cloudflare-exporter https://lablabs.github.io/cloudflare-exporter/
helm install cloudflare-exporter/cloudflare-exporter
```

## Docker

### Build

Images are available at [Github Container Registry](https://github.com/lablabs/cloudflare-exporter/pkgs/container/cloudflare_exporter)

To speed up multi-arch image builds and avoid burden of maintaining a Dockerfile, we are using [ko](https://ko.build/).

For more information on how to push the container image to your remote repository take a look at the [official docs](https://ko.build/get-started/).

To build localy use `KO_DOCKER_REPO=ko.local`

```shell
ko build .
```

### Run

Authenticating with email + API key:

```
docker run --rm -p 8080:8080 -e CF_API_KEY=${CF_API_KEY} -e CF_API_EMAIL=${CF_API_EMAIL} ghcr.io/lablabs/cloudflare_exporter
```

API token:

```
docker run --rm -p 8080:8080 -e CF_API_TOKEN=${CF_API_TOKEN} ghcr.io/lablabs/cloudflare_exporter
```

Configure zones and listening port:

```
docker run --rm -p 8080:8081 -e CF_API_TOKEN=${CF_API_TOKEN} -e CF_ZONES=zoneid1,zoneid2,zoneid3 -e LISTEN=:8081 ghcr.io/lablabs/cloudflare_exporter
```

Disable non-free metrics:

```
docker run --rm -p 8080:8080 -e CF_API_TOKEN=${CF_API_TOKEN} -e FREE_TIER=true ghcr.io/lablabs/cloudflare_exporter
```

Access help:

```
docker run --rm -p 8080:8080 -i ghcr.io/lablabs/cloudflare_exporter --help
```

## Development

### Building and Testing

This project uses a Makefile for common development tasks. Available targets:

- `make help` - Show all available targets
- `make fmt` - Format Go code using `go fmt` and `goimports`
- `make fmt-check` - Check if code is properly formatted (CI-friendly)
- `make lint` - Run golangci-lint to check code quality
- `make check` - Run all checks (formatting + linting)
- `make build` - Build the binary (runs checks first)
- `make test` - Run end-to-end tests
- `make clean` - Remove build artifacts

### Prerequisites

Make sure you have the following tools installed:

```bash
# Install goimports for import formatting
go install golang.org/x/tools/cmd/goimports@latest

# Install golangci-lint for linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Common Workflows

```bash
# Format code and build
make fmt && make build

# Run all code quality checks
make check

# Clean and rebuild
make clean build
```

## Contributing and reporting issues

Feel free to create an issue in this repository if you have questions, suggestions or feature requests.

### Validation, linters and pull-requests

We want to provide high quality code and modules. For this reason we are using
several [pre-commit hooks](.pre-commit-config.yaml) and
[GitHub Actions workflow](.github/workflows/golangci-lint.yml). A pull-request to the
master branch will trigger these validations and lints automatically. Please
check your code before you will create pull-requests. See
[pre-commit documentation](https://pre-commit.com/) and
[GitHub Actions documentation](https://docs.github.com/en/actions) for further
details.

## License

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

See [LICENSE](LICENSE) for full details.

    Licensed to the Apache Software Foundation (ASF) under one
    or more contributor license agreements.  See the NOTICE file
    distributed with this work for additional information
    regarding copyright ownership.  The ASF licenses this file
    to you under the Apache License, Version 2.0 (the
    "License"); you may not use this file except in compliance
    with the License.  You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing,
    software distributed under the License is distributed on an
    "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
    KIND, either express or implied.  See the License for the
    specific language governing permissions and limitations
    under the License.
