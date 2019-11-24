# go-infrabin

[![Build Status](https://travis-ci.org/maruina/go-infrabin.svg?branch=master)](https://travis-ci.org/maruina/go-infrabin)
[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)

[infrabin](https://github.com/maruina/infrabin) written in go.

## Endpoints

* `GET /healthcheck/liveness`
  * _returns_: the JSON `{"status": "liveness probe healthy"}` if healthy or the status code `503` if unhealthy.
