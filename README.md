# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)
[![Coverage Status](https://coveralls.io/repos/github/maruina/go-infrabin/badge.svg?branch=master)](https://coveralls.io/github/maruina/go-infrabin?branch=master)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes two ports:

* `8888` as a service port
* `8899` as the admin port, for liveness and readiness probes

To override the default values:

* _INFRABIN_MAX_DELAY_ to change the maximum value for the `/delay` endpoint. Default to 120.

## Installation

See the [README](./chart/go-infrabin/README.md).

## Environment variables

* `FAIL_ROOT_HANDLER`: if set, the `/` endpoint will return a 503. This is useful when doing a B/G deployment to test the failure and rollback scenario.

## Service Endpoints

* `GET /`
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

* `GET /delay/<seconds>`
  * _returns_: a JSON response

    ```json
    {
        "delay": "<seconds>"
    }
    ```

* `GET /headers`
  * _returns_: a JSON response with [HTTP headers](https://pkg.go.dev/net/http?tab=doc#Header)

    ```json
    {
        "headers": "<request headers>"
    }
    ```

* `GET /env/<env_var>`
  * _returns_: a JSON response with the requested `<env_var>` or `404` if the environment variable does not exist

    ```json
    {
        "env": {
            "<env_var>": "<env_var_value>"
        }
    }
    ```

## Admin Endpoints

* `GET /liveness`
  * _returns_: a JSON response if healthy or the status code `503` if unhealthy.

    ```json
    {
        "liveness": "pass"
    }
    ```

## Errors

* `400`
  * _returns_:

  ```json
    {
        "error": "<reason>"
    }
  ```


### Contributing

To build locally, ensure you have compiled the protocol schemas with:

```shell
protoc --proto_path=proto/ --go_out=plugins=grpc:pkg/infrabin --go_opt=paths=source_relative proto/infrabin.proto
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

To test grpc:

```shell
echo '{}' | evans -r cli call infrabin.Infrabin.Root
```
