# CloudFlare Prometheus exporter

[<img src="ll-logo.png">](https://lablabs.io/)

We help companies build, run, deploy and scale software and infrastructure by embracing the right technologies and principles. Check out our website at https://lablabs.io/

---

## Description

Prometheus exporter exposing Cloudflare Analytics dashboard data on per-zone basis.

## Configuration

The exporter can be configured using env variables

| **KEY** | **default** | **description** |
|-|-|-|
| `CF_API_KEY` | -- | API key |
| `CF_API_EMAIL` | -- | email associated with the API key (https://support.cloudflare.com/hc/en-us/articles/200167836-Managing-API-Tokens-and-Keys) |
| `ZONE_<NAME>` | -- | Zone ID. Add zones you want to scrape by adding env vars in this format. You can find the zone ids in Cloudflare dashboards |

## List of available metrics

```
# HELP cloudflare_zone_bandwidth_cached Cached bandwidth per zone in bytes
# HELP cloudflare_zone_bandwidth_content_type Bandwidth per zone per content type
# HELP cloudflare_zone_bandwidth_country Bandwidth per country per zone
# HELP cloudflare_zone_bandwidth_ssl_encrypted Encrypted bandwidth per zone in bytes
# HELP cloudflare_zone_bandwidth_total Total bandwidth per zone in bytes
# HELP cloudflare_zone_colocation_bandwidth_cached Total cached bandwidth per colocation
# HELP cloudflare_zone_colocation_bandwidth_total Total bandwidth per colocation
# HELP cloudflare_zone_colocation_requests_cached Total cached requests per colocation
# HELP cloudflare_zone_colocation_requests_country Requests per colocation per country
# HELP cloudflare_zone_colocation_requests_total Total requests per colocation
# HELP cloudflare_zone_colocation_response_status HTTP response status per colocation
# HELP cloudflare_zone_pageviews_total Pageviews per zone
# HELP cloudflare_zone_requests_cached Number of cached requests for zone
# HELP cloudflare_zone_requests_content_type Number of request for zone per content type
# HELP cloudflare_zone_requests_country Number of request for zone per country
# HELP cloudflare_zone_requests_ssl_encrypted Number of encrypted requests for zone
# HELP cloudflare_zone_requests_status Number of request for zone per HTTP status
# HELP cloudflare_zone_requests_total Number of requests for zone
# HELP cloudflare_zone_threats_country Threats per zone per country
# HELP cloudflare_zone_threats_total Threats per zone
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
