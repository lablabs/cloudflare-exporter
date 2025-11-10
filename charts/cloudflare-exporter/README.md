# CloudFlare Prometheus exporter Helm Chart

[<img src="ll-logo.png">](https://lablabs.io/)

We help companies build, run, deploy and scale software and infrastructure by embracing the right technologies and principles. Check out our website at https://lablabs.io/

---

## Description

A helm chart for [cloudflare-exporter](https://github.com/lablabs/cloudflare-exporter)

## Configuration


The following table lists the configurable parameters of the Cloudflare-exporter chart and their default values.

| Parameter                | Description             | Default        |
| ------------------------ | ----------------------- | -------------- |
| `replicaCount` |  | `1` |
| `image.repository` |  | `"ghcr.io/lablabs/cloudflare_exporter"` |
| `image.pullPolicy` |  | `"Always"` |
| `image.tag` |  | `"0.0.2"` |
| `env` | Environment variables for the exporter | `[]` |
| `secretRef` | The name of a secret with environment variables | `""` |
| `pathMetrics.enabled` | Enable path-level metrics grouped by zone, path, and HTTP status code | `false` |
| `pathMetrics.limit` | Maximum number of paths to track per zone (1-1000) | `100` |
| `imagePullSecrets` |  | `[]` |
| `nameOverride` |  | `""` |
| `fullnameOverride` |  | `""` |
| `podAnnotations` |  | `{}` |
| `podSecurityContext` |  | `{}` |
| `securityContext` |  | `{}` |
| `service.type` |  | `"ClusterIP"` |
| `service.port` |  | `8080` |
| `service.annotations.prometheus.io/probe` |  | `"true"` |
| `serviceAccount.enabled` |  | `false` |
| `serviceAccount.annotations` |  | `{}` |
| `serviceAccount.name` |  | `""` |
| `serviceMonitor.enabled` |  | `false` |
| `serviceMonitor.namespace` |  | `""` |
| `serviceMonitor.labels` |  | `{}` |
| `serviceMonitor.interval` |  | `"30s"` |
| `serviceMonitor.telemetryPath` |  | `"/metrics"` |
| `serviceMonitor.timeout` |  | `"10s"` |
| `serviceMonitor.relabelings` |  | `[]` |
| `serviceMonitor.targetLabels` |  | `[]` |
| `serviceMonitor.metricRelabelings` |  | `[]` |
| `resources` |  | `{}` |
| `nodeSelector` |  | `{}` |
| `tolerations` |  | `[]` |
| `affinity` |  | `{}` |

## Usage Examples

### Enable Path Metrics

To enable path-level metrics for tracking HTTP requests by zone, path, and status code:

```yaml
pathMetrics:
  enabled: true
  limit: 100
```

Note: Path metrics can generate significant cardinality depending on your traffic patterns. The `limit` parameter controls the maximum number of unique paths tracked per zone (top N by request count). This feature is not recommended for free-tier Cloudflare accounts.

### Configure with Custom Environment Variables

```yaml
env:
  - name: CF_API_TOKEN
    value: "your-api-token"
  - name: FREE_TIER
    value: "false"

pathMetrics:
  enabled: true
  limit: 50
```

## Contributing and reporting issues

Feel free to create an issue in this repository if you have questions, suggestions or feature requests.
