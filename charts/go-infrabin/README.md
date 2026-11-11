# go-infrabin

A Helm chart for Kubernetes installation of go-infrabin - infrabin written in go

## Installing the Chart

Install the chart from the `ghcr.io` OCI registry:

```console
helm install go-infrabin oci://ghcr.io/maruina/go-infrabin --version <chart-version>
```

## Developing

### Chart versions

The release workflow packages the chart and pushes it to `ghcr.io/maruina/go-infrabin` on every release. It syncs the chart `version` and `appVersion` with the released application version, so no manual packaging is needed.

### Testing

To test the Helm Chart in a KIND cluster

```console
kind create cluster
kubectl create ns infrabin
helm install quirky-walrus go-infrabin --namespace infrabin --set image.tag=latest --set image.pullPolicy=Always
helm test quirky-walrus -n infrabin
```

To test a change to the chart

```console
helm upgrade quirky-walrus go-infrabin --namespace infrabin --set image.tag=latest --set image.pullPolicy=Always
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| args.aWSMetadataEndpoint | string | `"http://169.254.169.254/latest/meta-data/"` |  |
| args.drainTimeout | string | `"15s"` |  |
| args.enableProxyEndpoint | bool | `false` |  |
| args.gRPCPort | int | `50051` |  |
| args.httpIdleTimeout | string | `"15s"` |  |
| args.httpReadHeaderTimeout | string | `"15s"` |  |
| args.httpReadTimeout | string | `"60s"` |  |
| args.httpWriteTimeout | string | `"121s"` |  |
| args.maxDelay | string | `"120s"` |  |
| args.promPort | int | `8887` |  |
| args.serverPort | int | `8888` |  |
| autoscaling.enabled | bool | `false` |  |
| autoscaling.maxReplicas | int | `100` |  |
| autoscaling.minReplicas | int | `1` |  |
| autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| extraEnv | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"maruina/go-infrabin"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| ingress.annotations | object | `{}` |  |
| ingress.enabled | bool | `false` |  |
| ingress.hosts[0].host | string | `"chart-example.local"` |  |
| ingress.hosts[0].paths | list | `[]` |  |
| ingress.tls | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| rbac.pspEnabled | bool | `false` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| tolerations | list | `[]` |  |
