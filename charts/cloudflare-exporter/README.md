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
| `image.repository` |  | `"lablabs/cloudflare_exporter"` |
| `image.pullPolicy` |  | `"Always"` |
| `image.tag` |  | `"0.0.2"` |
| `env` |  | `[]` |
| `imagePullSecrets` |  | `[]` |
| `nameOverride` |  | `""` |
| `fullnameOverride` |  | `""` |
| `podAnnotations` |  | `{}` |
| `podSecurityContext` |  | `{}` |
| `securityContext` |  | `{}` |
| `service.type` |  | `"ClusterIP"` |
| `service.port` |  | `8080` |
| `service.annotations.prometheus.io/probe` |  | `"true"` |
| `resources` |  | `{}` |
| `nodeSelector` |  | `{}` |
| `tolerations` |  | `[]` |
| `affinity` |  | `{}` |



## Contributing and reporting issues

Feel free to create an issue in this repository if you have questions, suggestions or feature requests.
