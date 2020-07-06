FROM golang:1.14.2-buster as builder
RUN apt-get update && \
    apt-get install -y zip && \
    mkdir protoc && \
    wget -O protoc/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.12.3/protoc-3.12.3-linux-x86_64.zip && \
    unzip protoc/protoc.zip -d protoc/ && \
    mv protoc/bin/protoc /usr/local/bin/ && \
    mv protoc/include /usr/local && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0
RUN wget -O /envoy-preflight https://github.com/monzo/envoy-preflight/releases/download/v1.0/envoy-preflight && \
    chmod +x /envoy-preflight
WORKDIR /go/src/go-infrabin
COPY . /go/src/go-infrabin
RUN go mod tidy && \
    go install \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
        google.golang.org/protobuf/cmd/protoc-gen-go \
        google.golang.org/grpc/cmd/protoc-gen-go-grpc && \
    make build && \
    make test


FROM gcr.io/distroless/base-debian10
COPY --from=builder /go/src/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
COPY --from=builder /envoy-preflight /envoy-preflight
ENTRYPOINT [ "/envoy-preflight", "/usr/local/bin/go-infrabin" ]
