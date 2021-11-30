ARG XX_VERSION=1.1.0

FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

# Docker buildkit multi-arch build requires golang alpine
FROM --platform=$BUILDPLATFORM golang:1.16-alpine as builder

# Copy the build utilities.
COPY --from=xx / /

ENV GRPC_HEALTH_PROBE_VERSION="v0.3.1"
ENV PROTOBUF_VERSION="3.15.5"
# build without specifing the arch
ENV CGO_ENABLED=0

RUN apk --no-cache add curl make && \
    curl -Lso /envoy-preflight "https://github.com/monzo/envoy-preflight/releases/download/v1.0/envoy-preflight" && \
    chmod +x /envoy-preflight && \
    curl -Lso /grpc_health_probe "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64" && \
    chmod +x /grpc_health_probe
WORKDIR /go/src/go-infrabin
COPY . /go/src/go-infrabin
RUN make tools && \
    make test && \
    xx-go build -o go-infrabin cmd/go-infrabin/main.go

FROM gcr.io/distroless/base-debian10@sha256:d2ce069a83a6407e98c7e0844f4172565f439dab683157bf93b6de20c5b46155
COPY --from=builder /go/src/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
COPY --from=builder /envoy-preflight /envoy-preflight
COPY --from=builder /grpc_health_probe /usr/local/bin/grpc_health_probe
ENTRYPOINT [ "/envoy-preflight", "/usr/local/bin/go-infrabin" ]
