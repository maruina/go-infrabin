// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: infrabin/v1/infrabin.proto

package infrabinv1connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v1 "github.com/maruina/go-infrabin/gen/infrabin/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// InfrabinServiceName is the fully-qualified name of the InfrabinService service.
	InfrabinServiceName = "infrabin.v1.InfrabinService"
)

// InfrabinServiceClient is a client for the infrabin.v1.InfrabinService service.
type InfrabinServiceClient interface {
	Headers(context.Context, *connect_go.Request[v1.HeadersRequest]) (*connect_go.Response[v1.HeadersResponse], error)
	Env(context.Context, *connect_go.Request[v1.EnvRequest]) (*connect_go.Response[v1.EnvResponse], error)
	Root(context.Context, *connect_go.Request[v1.RootRequest]) (*connect_go.Response[v1.RootResponse], error)
	Delay(context.Context, *connect_go.Request[v1.DelayRequest]) (*connect_go.Response[v1.DelayResponse], error)
}

// NewInfrabinServiceClient constructs a client for the infrabin.v1.InfrabinService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewInfrabinServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) InfrabinServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &infrabinServiceClient{
		headers: connect_go.NewClient[v1.HeadersRequest, v1.HeadersResponse](
			httpClient,
			baseURL+"/infrabin.v1.InfrabinService/Headers",
			opts...,
		),
		env: connect_go.NewClient[v1.EnvRequest, v1.EnvResponse](
			httpClient,
			baseURL+"/infrabin.v1.InfrabinService/Env",
			opts...,
		),
		root: connect_go.NewClient[v1.RootRequest, v1.RootResponse](
			httpClient,
			baseURL+"/infrabin.v1.InfrabinService/Root",
			opts...,
		),
		delay: connect_go.NewClient[v1.DelayRequest, v1.DelayResponse](
			httpClient,
			baseURL+"/infrabin.v1.InfrabinService/Delay",
			opts...,
		),
	}
}

// infrabinServiceClient implements InfrabinServiceClient.
type infrabinServiceClient struct {
	headers *connect_go.Client[v1.HeadersRequest, v1.HeadersResponse]
	env     *connect_go.Client[v1.EnvRequest, v1.EnvResponse]
	root    *connect_go.Client[v1.RootRequest, v1.RootResponse]
	delay   *connect_go.Client[v1.DelayRequest, v1.DelayResponse]
}

// Headers calls infrabin.v1.InfrabinService.Headers.
func (c *infrabinServiceClient) Headers(ctx context.Context, req *connect_go.Request[v1.HeadersRequest]) (*connect_go.Response[v1.HeadersResponse], error) {
	return c.headers.CallUnary(ctx, req)
}

// Env calls infrabin.v1.InfrabinService.Env.
func (c *infrabinServiceClient) Env(ctx context.Context, req *connect_go.Request[v1.EnvRequest]) (*connect_go.Response[v1.EnvResponse], error) {
	return c.env.CallUnary(ctx, req)
}

// Root calls infrabin.v1.InfrabinService.Root.
func (c *infrabinServiceClient) Root(ctx context.Context, req *connect_go.Request[v1.RootRequest]) (*connect_go.Response[v1.RootResponse], error) {
	return c.root.CallUnary(ctx, req)
}

// Delay calls infrabin.v1.InfrabinService.Delay.
func (c *infrabinServiceClient) Delay(ctx context.Context, req *connect_go.Request[v1.DelayRequest]) (*connect_go.Response[v1.DelayResponse], error) {
	return c.delay.CallUnary(ctx, req)
}

// InfrabinServiceHandler is an implementation of the infrabin.v1.InfrabinService service.
type InfrabinServiceHandler interface {
	Headers(context.Context, *connect_go.Request[v1.HeadersRequest]) (*connect_go.Response[v1.HeadersResponse], error)
	Env(context.Context, *connect_go.Request[v1.EnvRequest]) (*connect_go.Response[v1.EnvResponse], error)
	Root(context.Context, *connect_go.Request[v1.RootRequest]) (*connect_go.Response[v1.RootResponse], error)
	Delay(context.Context, *connect_go.Request[v1.DelayRequest]) (*connect_go.Response[v1.DelayResponse], error)
}

// NewInfrabinServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewInfrabinServiceHandler(svc InfrabinServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/infrabin.v1.InfrabinService/Headers", connect_go.NewUnaryHandler(
		"/infrabin.v1.InfrabinService/Headers",
		svc.Headers,
		opts...,
	))
	mux.Handle("/infrabin.v1.InfrabinService/Env", connect_go.NewUnaryHandler(
		"/infrabin.v1.InfrabinService/Env",
		svc.Env,
		opts...,
	))
	mux.Handle("/infrabin.v1.InfrabinService/Root", connect_go.NewUnaryHandler(
		"/infrabin.v1.InfrabinService/Root",
		svc.Root,
		opts...,
	))
	mux.Handle("/infrabin.v1.InfrabinService/Delay", connect_go.NewUnaryHandler(
		"/infrabin.v1.InfrabinService/Delay",
		svc.Delay,
		opts...,
	))
	return "/infrabin.v1.InfrabinService/", mux
}

// UnimplementedInfrabinServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedInfrabinServiceHandler struct{}

func (UnimplementedInfrabinServiceHandler) Headers(context.Context, *connect_go.Request[v1.HeadersRequest]) (*connect_go.Response[v1.HeadersResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("infrabin.v1.InfrabinService.Headers is not implemented"))
}

func (UnimplementedInfrabinServiceHandler) Env(context.Context, *connect_go.Request[v1.EnvRequest]) (*connect_go.Response[v1.EnvResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("infrabin.v1.InfrabinService.Env is not implemented"))
}

func (UnimplementedInfrabinServiceHandler) Root(context.Context, *connect_go.Request[v1.RootRequest]) (*connect_go.Response[v1.RootResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("infrabin.v1.InfrabinService.Root is not implemented"))
}

func (UnimplementedInfrabinServiceHandler) Delay(context.Context, *connect_go.Request[v1.DelayRequest]) (*connect_go.Response[v1.DelayResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("infrabin.v1.InfrabinService.Delay is not implemented"))
}
