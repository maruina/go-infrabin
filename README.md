# go-infrabin

[infrabin](https://github.com/maruina/infrabin) written in go.

## Endpoints

* `GET /healthcheck/liveness`
  * _returns_: the JSON `{"status": "liveness probe healthy"}` if healthy or the status code `503` if unhealthy.
