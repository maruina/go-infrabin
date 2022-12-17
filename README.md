# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes one ports:

* `8888` as a HTTP/gRPC port

## Installation with Helm

See the [Helm Chart README](./chart/go-infrabin/README.md).

## Usage

```console
Usage:
  go-infrabin [flags]

Flags:
      --addr string                    TCP address for the server to listen on (default ":8888")
      --drain-timeout duration         Drain timeout to wait for in-flight connections to terminate before closing the connection (default 1m0s)
  -h, --help                           help for go-infrabin
      --idle-timeout duration          HTTP idle timeout (default 15s)
      --read-header-timeout duration   HTTP read header timeout (default 15s)
      --read-timeout duration          HTTP read timeout (default 1m0s)
      --write-timeout duration         HTTP write timeout (default 15s)
```

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
  * _NOTE_: the target endpoint **MUST** provide a JSON response
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

* _grpc_: `infrabin.Infrabin.AWSMetadata` _rest_: `GET /aws/metadata/<path>`
  * _NOTE_: `--enable-proxy-endpoint` must be set
  * _grpc request_

  ```text
  message AWSRequest {
      string path = 1;
  }
  ```

  * _returns_: JSON of AWS GET call

* _grpc_: `infrabin.Infrabin.Any` _rest_: `GET /any/<path>`

  ```text
  message Any {
      string path = 1;
  }
  ```

  * _returns_: JSON of the requested path

* _grpc_: `infrabin.Infrabin.AWSAssume` _rest_: `GET /aws/assume/<role>`
  * _grpc request_

  ```text
  message AWSAssume {
      string role = 1;
  }
  ```

  * _returns_: JSON with the AssumedRoleId from AWS

  ```json
  {
    "assumedRoleId":"AROAITQZVNCXXXXXXXXXX:aws-assume-session-go-infrabin"
  }
  ```

* _grpc_: `infrabin.Infrabin.AWSGetCallerIdentity` _rest_: `GET /aws/get-caller-identity`
  * _grpc request_

  ```text
  message Empty {}
  ```

  * _returns_: JSON with the GetCallerIdentity output from AWS

  ```json
  {
    "getCallerIdentity": {
      "account": "123456789",
      "arn": "arn:aws:sts::1234546789:assumed-role/foo/bar",
      "user_id": "AROAITQZVNCVSVXXXXXX:foo"
    }
  }
  ```

* _grpc_: `infrabin.Infrabin.Intermittent` _rest_: `GET /intermittent`
  * _grpc request_

  ```text
  message Empty {}
  ```

  * _returns_: JSON with the remaining errors and then the configured `--intermittent-errors` flag

  ```json
  {"code":14, "message":"2 errors left"}
  ```

  ```json
  {"intermittent":{"consecutive_errors":2}
  ```

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
