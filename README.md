# go-infrabin

[![Build Status](https://travis-ci.org/maruina/go-infrabin.svg?branch=master)](https://travis-ci.org/maruina/go-infrabin)
[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)
[![Coverage Status](https://coveralls.io/repos/github/maruina/go-infrabin/badge.svg?branch=master)](https://coveralls.io/github/maruina/go-infrabin?branch=master)

[infrabin](https://github.com/maruina/infrabin) written in go.

## Usage

`go-infrabin` exposes two ports:

* `8888` as a service port
* `8899` as the admin port, for livensss and readiness probes

To override the default values:

* _INFRABIN_MAX_DELAY_ to change the maximum value for the `/delay` endpoint. Default to 120.

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
