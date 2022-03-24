// +build tools

// This file ensures tool dependencies are kept in sync. This is the
// recommended way of doing this according to
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// To install the following tools at the version used by this repo run:
// $ make tools
// or
// $ go generate -tags tools tools/tools.go

package tools

//go:generate go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
//go:generate go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
//go:generate go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go
import (
_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"

_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"

_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
