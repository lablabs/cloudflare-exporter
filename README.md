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

## Helm chart repository

To deploy the exporter into Kubernetes, we recommend to use our manager Helm repository at

```
helm repo add cloudflare-exporter https://lablabs.github.io/cloudflare-exporter/
helm install cloudflare-exporter/cloudflare-exporter
```

## Configuration

The exporter can be configured using env variables

| **KEY** | **description** |
|-|-|
| `LISTEN` |  listen on addr:port ( default :8080), omit addr to listen on all interfaces |
| `METRICS_PATH` |  path for metrics, default /metrics |
| `CF_API_KEY` |  API key |
| `CF_API_EMAIL` |  email associated with the API key (https://support.cloudflare.com/hc/en-us/articles/200167836-Managing-API-Tokens-and-Keys) |
| `CF_API_TOKEN` |  API authentification token (https://developers.cloudflare.com/analytics/graphql-api/getting-started/authentication/api-token-auth) |
| `ZONE_<NAME>` |  DEPRECATED (optional) Zone ID. Add zones you want to scrape by adding env vars in this format. You can find the zone ids in Cloudflare dashboards. |
| `CF_ZONES` |  (Optional) cloudflare zones to export, comma delimited list of zone ids, if not set, all zones from account are exported |
Defaults to all zones. |

Another configuration options are command line flags, same as environmental variables but lowercase, zones are not supported as flag, see ./cloudflare_exporter --help

```
  -cf_api_email="": cloudflare api email, works with api_key flag
  -cf_api_key="": cloudflare api key, works with api_email flag
  -cf_api_token="": cloudflare api token (preferred)
  -cf_zones="": cloudflare zones to export, comma delimited list
  -listen=":8080": listen on addr:port ( default :8080), omit addr to listen on all interfaces
  -metrics_path="/metrics": path for metrics, default /metrics
```

### Changes in in version 0.0.5+

## Authentication

From version 0.0.5 onward authentication using Bearer token is supported. Authentication using API key and email will continue working in later versions as well. The token authentication method is preferred.

## Zone filtering

The original method of zone filtering by using env variables `ZONE_<name>` is now deprecated. Zones can be filtered by using `CF_ZONES` env variable and setting the value as list of zones separated by a comma (CF_ZONES=zone1,zone2,zone3).



## List of available metrics

```
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


## Docker



### Build

Images are available at [Dockerhub](https://hub.docker.com/r/lablabs/cloudflare_exporter)

```
docker build -t lablabs/cloudflare_exporter .
```

### Run

```
docker run --rm -p 8080:8080 -e CF_API_KEY=${CF_API_KEY} -e CF_API_EMAIL=${CF_API_EMAIL} lablabs/cloudflare_exporter
```
or
```
docker run --rm -p 8080:8080 -e CF_API_TOKEN=${CF_API_TOKEN} lablabs/cloudflare_exporter
```
or example with selected zones and listen port
```
docker run --rm -p 8080:8081 -e CF_API_TOKEN=${CF_API_TOKEN} -e CF_ZONES=zoneid1,zoneid2,zoneid3 -e LISTEN=:8081 lablabs/cloudflare_exporter
```
help
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
