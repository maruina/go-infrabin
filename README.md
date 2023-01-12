# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes expose port `8888` as a HTTP/gRPC port using the [connect](https://connect.build/) protocol.

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

## Examples

```console
❯ curl -XPOST --header "Content-Type: application/json" --data '{}' http://localhost:8888/infrabin.v1.InfrabinService/Root
{"hostname":"XYZ"}%

❯ curl -XPOST --header "Content-Type: application/json" --data '{"method": "GET", "url":"https://google.com"}' http://localhost:8888/infrabin.v1.InfrabinService/Proxy
{"statusCode":200,"headers":{"Alt-Svc":"h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000,h3-Q050=\":443\"; ma=2592000,h3-Q046=\":443\"; ma=2592000,h3-Q043=\":443\"; ma=2592000,quic=\":443\"; ma=2592000; v=\"46,43\"","Cache-Control":"private, max-age=0","Content-Type":"text/html; charset=ISO-8859-1","Cross-Origin-Opener-Policy-Report-Only":"same-origin-allow-popups; report-to=\"gws\"","Date":"Sun, 18 Dec 2022 14:16:56 GMT","Expires":"-1","P3p":"CP=\"This is not a P3P policy! See g.co/p3phelp for more info.\"","Report-To":"{\"group\":\"gws\",\"max_age\":2592000,\"endpoints\":[{\"url\":\"https://csp.withgoogle.com/csp/report-to/gws/other\"}]}","Server":"gws","Set-Cookie":"CONSENT=PENDING+491; expires=Tue, 17-Dec-2024 14:16:56 GMT; path=/; domain=.google.com; Secure","X-Frame-Options":"SAMEORIGIN","X-Xss-Protection":"0"}}%
```

## Contributing

Requirements:

* go v1.19+
* [buf](https://docs.buf.build/installation)

```console
❯ make help
all                            Build, test, and lint (default)
build                          Build all packages
clean                          Delete intermediate build artifacts
generate                       Regenerate code
help                           Describe useful make targets
lint                           Lint Go and protobuf
lintfix                        Automatically fix some lint errors
test                           Run unit tests
upgrade                        Upgrade dependencies
```
