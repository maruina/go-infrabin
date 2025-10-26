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
* `--drain-timeout`: Drain timeout (default `15s`)
* `--egress-timeout`: Timeout for egress HTTP/HTTPS requests (default `3s`)
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

* `POD_NAME` or `K8S_POD_NAME`: Kubernetes pod name
* `POD_NAMESPACE` or `K8S_NAMESPACE`: Kubernetes namespace
* `POD_IP` or `K8S_POD_IP`: Pod IP address
* `NODE_NAME` or `K8S_NODE_NAME`: Kubernetes node name
* `CLUSTER_NAME` or `K8S_CLUSTER_NAME`: Kubernetes cluster name
* `REGION`, `AWS_REGION`, or `FUNCTION_REGION`: Cloud region

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
