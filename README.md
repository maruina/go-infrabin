# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)
[![Coverage Status](https://coveralls.io/repos/github/maruina/go-infrabin/badge.svg?branch=master)](https://coveralls.io/github/maruina/go-infrabin?branch=master)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes three ports:

* `8888` as a http rest port
* `8887` as http Prometheus metrics port (endpoint: `/metrics`)
* `50051` as a grpc port

## Installation

See the [README](./chart/go-infrabin/README.md).

## Command line flags

* `--aws-metadata-endpoint`: AWS Metadata Endpoint (default `http://169.254.169.254/latest/meta-data/`)
* `--crossaz-label-selector`: Label selector for discovering pods in `/crossaz` endpoint (default `"app.kubernetes.io/name=go-infrabin"`)
* `--crossaz-timeout`: Timeout for cross-AZ connectivity tests (default `3s`)
* `--drain-timeout`: Drain timeout (default `15s`)
* `--egress-timeout`: Timeout for egress HTTP/HTTPS requests (default `3s`)
* `--enable-crossaz-endpoint`: When enabled allows `/crossaz` endpoint (requires Kubernetes RBAC permissions)
* `--enable-proxy-endpoint`: When enabled allows `/proxy` and `/aws` endpoints
* `--proxy-allow-regexp`: Regular expression to allow URL called by the `/proxy` endpoint (default `".*"`)
* `--intermittent-errors`: Number of consecutive 503 errors before returning 200 when calling the `/intermittent` endpoint (default `2`)
* `--grpc-host`: gRPC host (default `0.0.0.0`)
* `--grpc-port`: gRPC port (default `50051`)
* `-h`, `--help`: Help for go-infrabin
* `--http-idle-timeout`: HTTP idle timeout (default `15s`)
* `--http-read-header-timeout`: HTTP read header timeout (default `15s`)
* `--http-read-timeout`: HTTP read timeout (default `1m0s`)
* `--http-write-timeout`: HTTP write timeout (default `2m1s`)
* `--max-delay duration`: Maximum delay (default `2m0s`)
* `--prom-host`: Prometheus metrics host (default `0.0.0.0`)
* `--prom-port`: Prometheus metrics port (default `8887`)
* `--server-host`: HTTP server host (default `0.0.0.0`)
* `--server-port`: HTTP server port (default `8888`)

## Environment variables

* `FAIL_ROOT_HANDLER`: if set, the `/` endpoint will return a 503. This is useful when doing a B/G deployment to test the failure and rollback scenario.

### Kubernetes Environment Variables

The following environment variables are used to populate the Kubernetes information in the root endpoint response:

* `POD_NAME` or `K8S_POD_NAME`: Kubernetes pod name (also required for `/crossaz` endpoint)
* `POD_NAMESPACE` or `K8S_NAMESPACE`: Kubernetes namespace
* `POD_IP` or `K8S_POD_IP`: Pod IP address
* `NODE_NAME` or `K8S_NODE_NAME`: Kubernetes node name
* `CLUSTER_NAME` or `K8S_CLUSTER_NAME`: Kubernetes cluster name
* `REGION`, `AWS_REGION`, or `FUNCTION_REGION`: Cloud region
* `AVAILABILITY_ZONE`: Availability zone (required for `/crossaz` endpoint)

## API Documentation

For comprehensive API documentation, the service exposes an OpenAPI (Swagger) specification at:

* **OpenAPI Spec**: `GET /openapi.json` (available on port `8888`)

This specification is auto-generated from the protobuf definitions and includes:
- Complete endpoint documentation with descriptions
- Request/response schemas
- gRPC and REST API mappings
- Field descriptions and validation rules

You can use tools like [Swagger UI](https://swagger.io/tools/swagger-ui/), [Redoc](https://github.com/Redocly/redoc), or [Postman](https://www.postman.com/) to explore the API interactively.

### Quick Reference

The service exposes both gRPC (`infrabin.Infrabin`) and REST endpoints:

| Endpoint | Description |
|----------|-------------|
| `GET /` | Service information (hostname, Kubernetes metadata) |
| `GET /delay/{seconds}` | Artificial delay for testing |
| `GET /headers` | Echo request headers |
| `GET /env/{env_var}` | Retrieve environment variable |
| `POST /proxy` | Proxy HTTP requests (requires `--enable-proxy-endpoint`) |
| `GET /aws/metadata/{path}` | Query AWS metadata service (requires `--enable-proxy-endpoint`) |
| `GET /aws/assume/{role}` | Assume AWS IAM role |
| `GET /aws/get-caller-identity` | AWS STS GetCallerIdentity |
| `GET /any/{path}` | Wildcard path echo |
| `GET /intermittent` | Simulate intermittent failures |
| `GET /bytes/{number}` | Generate random bytes |
| `GET /egress/dns/{host}` | Test DNS resolution for a given hostname |
| `GET /egress/http/{target}` | Test HTTP connectivity (port 80 by default) |
| `GET /egress/https/{target}` | Test HTTPS connectivity with certificate verification (port 443) |
| `GET /egress/https/insecure/{target}` | Test HTTPS connectivity without certificate verification |
| `POST /healthcheck/liveness/{status}` | Set liveness probe status (`pass` or `fail`) |
| `POST /healthcheck/readiness/{status}` | Set readiness probe status (`pass` or `fail`) |
| `GET /crossaz` | Test cross-availability-zone connectivity (requires `--enable-crossaz-endpoint`) |

#### Egress Endpoints

The egress endpoints support testing network connectivity and DNS resolution:

**DNS Resolution**:
```bash
# Test DNS resolution using system DNS
curl http://localhost:8888/egress/dns/google.com

# Test with custom DNS server (format: hostname@dns_server:port)
curl http://localhost:8888/egress/dns/google.com@8.8.8.8:53

# Test with custom DNS server (port defaults to 53)
curl http://localhost:8888/egress/dns/google.com@1.1.1.1

# Returns resolved IP addresses and duration
```

**HTTP/HTTPS Connectivity**:
```bash
# Test HTTP connection (default port 80)
curl http://localhost:8888/egress/http/example.com

# Test HTTPS connection with certificate verification (default port 443)
curl http://localhost:8888/egress/https/example.com

# Test with custom port
curl http://localhost:8888/egress/http/example.com:8080

# Test with custom DNS server (format: target@dns_server:port)
curl http://localhost:8888/egress/http/example.com@8.8.8.8:53

# Test HTTPS without certificate verification (useful for self-signed certs)
curl http://localhost:8888/egress/https/insecure/self-signed.example.com
```

All egress endpoints return timing information even on failure, which is useful for diagnosing network issues. The timeout can be configured with `--egress-timeout` (default: 3s).

#### Health Check Endpoints

The health check endpoints allow you to dynamically control the liveness and readiness probe status, useful for testing Kubernetes probe behavior and failover scenarios:

**gRPC Health Checks**:

The service implements the standard `grpc.health.v1.Health` service with two registered service names:
- `liveness` - for liveness probe checks
- `readiness` - for readiness probe checks

```bash
# Check liveness status via gRPC
grpcurl -plaintext -d '{"service": "liveness"}' localhost:50051 grpc.health.v1.Health/Check

# Check readiness status via gRPC
grpcurl -plaintext -d '{"service": "readiness"}' localhost:50051 grpc.health.v1.Health/Check
```

**HTTP Health Check Control**:

```bash
# Set liveness to healthy (SERVING)
curl -X POST http://localhost:8888/healthcheck/liveness/pass

# Set liveness to unhealthy (NOT_SERVING)
curl -X POST http://localhost:8888/healthcheck/liveness/fail

# Set readiness to ready (SERVING)
curl -X POST http://localhost:8888/healthcheck/readiness/pass

# Set readiness to not ready (NOT_SERVING)
curl -X POST http://localhost:8888/healthcheck/readiness/fail
```

**Kubernetes Integration**:

```yaml
# Using gRPC probes (Kubernetes 1.24+)
livenessProbe:
  grpc:
    service: liveness
    port: 50051
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  grpc:
    service: readiness
    port: 50051
  initialDelaySeconds: 5
  periodSeconds: 10

# Using HTTP probes (via grpc-gateway)
livenessProbe:
  httpGet:
    path: /check?service=liveness
    port: 8888
  initialDelaySeconds: 5
  periodSeconds: 10
```

By default, both liveness and readiness are set to healthy (SERVING) on startup. Use the control endpoints to simulate probe failures and test your orchestration behavior.

#### CrossAZ Endpoint

The `/crossaz` endpoint helps test cross-availability-zone connectivity in Kubernetes clusters. It automatically discovers pods running in different availability zones and tests HTTP connectivity between them, which is useful for:
- Identifying cross-AZ network misconfigurations
- Monitoring cross-AZ connectivity health
- Validating network policies across zones
- Testing zone-aware load balancing

**Prerequisites**:
1. Enable the endpoint with `--enable-crossaz-endpoint`
2. Configure RBAC permissions (see Helm chart configuration below)
3. Set the `AVAILABILITY_ZONE` and `POD_NAME` environment variables via Kubernetes Downward API

**Usage**:
```bash
# Test cross-AZ connectivity
curl http://localhost:8888/crossaz
```

**Response includes**:
- Current pod's AZ and name
- Discovered pods grouped by availability zone
- Connectivity test results for each cross-AZ pod
- Summary statistics (total pods, AZs, successful/failed tests)

**Prometheus Metrics**:
```
# Total number of cross-AZ connectivity tests
crossaz_tests_total{source_az="us-east-1a",target_az="us-east-1b",result="success"}

# Duration of connectivity tests in milliseconds
crossaz_test_duration_milliseconds{source_az="us-east-1a",source_pod="pod-1",target_az="us-east-1b",destination_pod="pod-2"}

# Number of pods discovered per availability zone
crossaz_pods_discovered{az="us-east-1a"}
```

**Helm Chart Configuration**:
```yaml
args:
  enableCrossAZEndpoint: true
  crossAZTimeout: 3s
  crossAZLabelSelector: "app.kubernetes.io/name=go-infrabin"

rbac:
  crossAZ:
    enabled: true
```

When `enableCrossAZEndpoint` is true, the Helm chart automatically:
- Sets replica count to 3 (unless autoscaling is enabled)
- Adds topology spread constraints to distribute pods across availability zones (maxSkew: 1)
- Configures Downward API environment variables (POD_NAME, AVAILABILITY_ZONE)

The endpoint requires both namespace-scoped (pods) and cluster-scoped (nodes) RBAC permissions to extract availability zone information from node labels.

For detailed request/response schemas and field descriptions, see `/openapi.json`.

### Contributing

To build locally, ensure you have compiled the protocol schemas. You
will need the `protoc` binary which can install by following
[these instructions][protoc] or if using Homebrew

```shell
brew install protobuf
```

You will also need to protoc go plugins for protofuf, grpc, and
grpc-gateway. `go mod tidy` will fetch the versions specified in
`go.mod`, and `go install` will install that version.

```shell
go get -t -v -d ./...
go install \
  github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
  github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
  google.golang.org/protobuf/cmd/protoc-gen-go \
  google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

`make run` will compile the protocol buffers, or you can run:

```shell
make protoc
```

To run the tests:

```shell
make test
```

To run the server locally:

```shell
make run
```

To test http:

```shell
http localhost:8888/
```

To test grpc, use your favourite grpc tool like [evans][evans]:

```shell
echo '{}' | evans -r cli call infrabin.Infrabin.Root
```

[protoc]: https://grpc.io/docs/languages/go/quickstart/#prerequisites
[evans]: https://github.com/ktr0731/evans/
