# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)
[![Coverage Status](https://coveralls.io/repos/github/maruina/go-infrabin/badge.svg?branch=master)](https://coveralls.io/github/maruina/go-infrabin?branch=master)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes three ports:

* `8888` as a http rest port
* `8887` as http Prometheus port
* `50051` as a grpc port

## Installation

See the [README](./chart/go-infrabin/README.md).

## Command line flags

* `--admin-host`: HTTP server host (default `0.0.0.0`)
* `--admin-port`: HTTP server port (default `8889`)
* `--aws-metadata-endpoint`: AWS Metadata Endpoint (default `http://169.254.169.254/latest/meta-data/`)
* `--drain-timeout`: Drain timeout (default `15s`)
* `--enable-proxy-endpoint`: When enabled allows `/proxy` and `/aws` endpoints
* `--grpc-host`: gRPC host (default `0.0.0.0`)
* `--grpc-port`: gRPC port (default `50051`)
* `-h`, `--help`: help for go-infrabin
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

* `INFRABIN_MAX_DELAY`: to change the maximum value for the `/delay` endpoint. Default to 120.
* `FAIL_ROOT_HANDLER`: if set, the `/` endpoint will return a 503. This is useful when doing a B/G deployment to test the failure and rollback scenario.

## Service Endpoints

* _grpc_: `infrabin.Infrabin.Root` _rest_: `GET /`
  * _grpc request_

  ```text
  message Empty {}
  ```

  * _returns_: a JSON response

  ```json
  {
      "hostname": "<hostname>",
      "kubernetes": {
          "pod_name": "<pod_name>",
          "namespace": "<namespace>",
          "pod_ip": "<pod_ip>",
          "node_name": "<node_name>"
      }
  }
  ```

* _grpc_: `infrabin.Infrabin.Delay` _rest_: `GET /delay/<seconds>`
  * _grpc request_

  ```text
  message DelayRequest {
    int32 duration = 1;
  }
  ```

  * _returns_: a JSON response

  ```json
  {
      "delay": "<seconds>"
  }
  ```

* _grpc_: `infrabin.Infrabin.Headers` _rest_: `GET /headers`

  * _grpc request_

  ```text
  message HeadersRequest {
      map<string, string> headers = 1;
  }
  ```

  * _returns_: a JSON response with [HTTP headers](https://pkg.go.dev/net/http?tab=doc#Header)

  ```json
  {
      "headers": "<request headers>"
  }
  ```

* _grpc_: `infrabin.Infrabin.Env` _rest_: `GET /env/<env_var>`
  * _grpc request_

  ```text
  message EnvRequest {
      string env_var = 1;
  }
  ```

  * _returns_: a JSON response with the requested `<env_var>` or `404` if the environment variable does not exist

  ```json
  {
      "env": {
          "<env_var>": "<env_var_value>"
      }
  }
  ```

* _grpc_: `infrabin.Infrabin.Proxy` _rest_: `GET /proxy`
  * _NOTE_: `--enable-proxy-endpoint` must be set
  * _grpc request_

  ```text
  message ProxyRequest {
      string method = 1;
      string url = 2;
      google.protobuf.Struct body = 3;
      map<string, string> headers = 4;
  }
  ```

  * _returns_: JSON of proxied request

* _grpc_: `infrabin.Infrabin.AWS` _rest_: `GET /aws/<path>`
  * _NOTE_: `--enable-proxy-endpoint` must be set
  * _grpc request_

  ```text
  message AWSRequest {
      string path = 1;
  }
  ```

  * _returns_: JSON of AWS GET call

## Errors

When calling the http endpoint, errors are mapped by `grpc-gateway`. They have the following format:

* _returns_:

```json
{
    "code": 3,
    "message": "type mismatch, parameter: duration, error: strconv.ParseInt: parsing \"21asd\": invalid syntax"
}
```

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
