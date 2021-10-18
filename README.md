# CloudFlare Prometheus exporter
[<img src="ll-logo.png">](https://lablabs.io/)

We help companies build, run, deploy and scale software and infrastructure by embracing the right technologies and principles. Check out our website at https://lablabs.io/

---

## Description
Prometheus exporter exposing Cloudflare Analytics dashboard data on a per-zone basis, as well as Worker metrics.
The exporter is also able to scrape Zone metrics by Colocations (https://www.cloudflare.com/network/).

## Grafana Dashboard
![Dashboard](https://i.ibb.co/HDsqDF1/cf-exporter.png)

Our public dashboard is available at https://grafana.com/grafana/dashboards/13133


## Authentication
Authentication towards the Cloudflare API can be done in two ways:

### API token
The preferred way of authenticating is with an API token, for which the scope can be configured at the Cloudflare
dashboard.

Required authentication scopes:
- `Analytics:Read` is required for zone-level metrics
- `Account.Account Analytics:Read` is required for Worker metrics
- `Account Settings:Read` is required for Worker metrics (for listing accessible accounts, scraping all available
  Workers included in authentication scope)

To authenticate this way, only set `CF_API_TOKEN` (omit `CF_API_EMAIL` and `CF_API_KEY`)

### User email + API key
To authenticate with user email + API key, use the `Global API Key` from the Cloudflare dashboard.
Beware that this key authenticates with write access to every Cloudflare resource.

To authenticate this way, set both `CF_API_KEY` and `CF_API_EMAIL`.

## Configuration
The exporter can be configured using env variables or command flags.

| **KEY** | **description** |
|-|-|
| `CF_API_EMAIL` |  user email (see https://support.cloudflare.com/hc/en-us/articles/200167836-Managing-API-Tokens-and-Keys) |
| `CF_API_KEY` |  API key associated with email (`CF_API_EMAIL` is required if this is set)|
| `CF_API_TOKEN` |  API authentication token (recommended before API key + email. Version 0.0.5+. see https://developers.cloudflare.com/analytics/graphql-api/getting-started/authentication/api-token-auth) |
| `CF_ZONES` |  (Optional) cloudflare zones to export, comma delimited list of zone ids. If not set, all zones from account are exported |
| `FREE_TIER` | (Optional) scrape only metrics included in free plan. Accepts `true` or `false`, default `false`. |
| `LISTEN` |  listen on addr:port (default `:8080`), omit addr to listen on all interfaces |
| `METRICS_PATH` |  path for metrics, default `/metrics` |
| `SCRAPE_DELAY` | scrape delay in seconds, default `300`
| `ZONE_<NAME>` |  `DEPRECATED since 0.0.5` (optional) Zone ID. Add zones you want to scrape by adding env vars in this format. You can find the zone ids in Cloudflare dashboards. |

Corresponding flags:
```
  -cf_api_email="": cloudflare api email, works with api_key flag
  -cf_api_key="": cloudflare api key, works with api_email flag
  -cf_api_token="": cloudflare api token (version 0.0.5+, preferred)
  -cf_zones="": cloudflare zones to export, comma delimited list
  -free_tier=false: scrape only metrics included in free plan, default false
  -listen=":8080": listen on addr:port ( default :8080), omit addr to listen on all interfaces
  -metrics_path="/metrics": path for metrics, default /metrics
  -scrape_delay=300: scrape delay in seconds, defaults to 300
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
# HELP cloudflare_zone_pageviews_total Pageviews per zone
# HELP cloudflare_zone_requests_cached Number of cached requests for zone
# HELP cloudflare_zone_requests_content_type Number of request for zone per content type
# HELP cloudflare_zone_requests_country Number of request for zone per country
# HELP cloudflare_zone_requests_origin_status_country_host Count of not cached requests for zone per origin HTTP status per country per host
# HELP cloudflare_zone_requests_ssl_encrypted Number of encrypted requests for zone
# HELP cloudflare_zone_requests_status Number of request for zone per HTTP status
# HELP cloudflare_zone_requests_status_country_host Count of requests for zone per edge HTTP status per country per host
# HELP cloudflare_zone_requests_total Number of requests for zone
# HELP cloudflare_zone_threats_country Threats per zone per country
# HELP cloudflare_zone_threats_total Threats per zone
# HELP cloudflare_zone_uniques_total Uniques per zone
```

## Helm chart repository
To deploy the exporter into Kubernetes, we recommend using our manager Helm repository:

```
helm repo add cloudflare-exporter https://lablabs.github.io/cloudflare-exporter/
helm install cloudflare-exporter/cloudflare-exporter
```

## Docker
### Build
Images are available at [Dockerhub](https://hub.docker.com/r/lablabs/cloudflare_exporter)

```
docker build -t lablabs/cloudflare_exporter .
```

### Run
Authenticating with email + API key:
```
docker run --rm -p 8080:8080 -e CF_API_KEY=${CF_API_KEY} -e CF_API_EMAIL=${CF_API_EMAIL} lablabs/cloudflare_exporter
```

API token:
```
docker run --rm -p 8080:8080 -e CF_API_TOKEN=${CF_API_TOKEN} lablabs/cloudflare_exporter
```

Configure zones and listening port:
```
docker run --rm -p 8080:8081 -e CF_API_TOKEN=${CF_API_TOKEN} -e CF_ZONES=zoneid1,zoneid2,zoneid3 -e LISTEN=:8081 lablabs/cloudflare_exporter
```

Disable non-free metrics:
```
docker run --rm -p 8080:8080 -e CF_API_TOKEN=${CF_API_TOKEN} -e FREE_TIER=true lablabs/cloudflare_exporter
```

Access help:
```
docker run --rm -p 8080:8080 -i lablabs/cloudflare_exporter --help
```

## Contributing and reporting issues
Feel free to create an issue in this repository if you have questions, suggestions or feature requests.

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
