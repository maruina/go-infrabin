FROM golang:1.18-buster as builder
ENV GRPC_HEALTH_PROBE_VERSION="v0.3.1"
ENV PROTOBUF_VERSION="3.20.1"

RUN apt-get update && \
    apt-get install -y --no-install-recommends unzip=6.0-23+deb10u2 && \
    mkdir protoc && \
    curl -Lso protoc/protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-linux-x86_64.zip" && \
    unzip protoc/protoc.zip -d protoc/ && \
    mv protoc/bin/protoc /usr/local/bin/ && \
    mv protoc/include /usr/local && \
    curl -Lso /envoy-preflight "https://github.com/monzo/envoy-preflight/releases/download/v1.0/envoy-preflight" && \
    chmod +x /envoy-preflight && \
    curl -Lso /grpc_health_probe "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64" && \
    chmod +x /grpc_health_probe
WORKDIR /go/src/go-infrabin
COPY . /go/src/go-infrabin
RUN make tools && \
    make test && \
    make build

FROM gcr.io/distroless/base-debian10@sha256:fc685e6d0c65fb8255840a6f235182a4ff42fdc4860be69ecf272443edc00f1e
COPY --from=builder /go/src/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
COPY --from=builder /envoy-preflight /envoy-preflight
COPY --from=builder /grpc_health_probe /usr/local/bin/grpc_health_probe
ENTRYPOINT [ "/envoy-preflight", "/usr/local/bin/go-infrabin" ]
