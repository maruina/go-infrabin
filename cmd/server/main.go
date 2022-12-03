package main

import (
	"context"
	"net/http"

	"github.com/bufbuild/connect-go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	infrabinv1 "github.com/maruina/go-infrabin/gen/infrabin/v1"        // generated by protoc-gen-go
	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect" // generated by protoc-gen-connect-go
)

type InfrabinServer struct{}

func (s *InfrabinServer) Headers(ctx context.Context, req *connect.Request[infrabinv1.HeadersRequest],
) (*connect.Response[infrabinv1.HeadersResponse], error) {
	res := connect.NewResponse(&infrabinv1.HeadersResponse{
		Headers: map[string]string{},
	})
	res.Header().Set("Greet-Version", "v1")
	return res, nil
}

func main() {
	infraServer := &InfrabinServer{}
	mux := http.NewServeMux()
	path, handler := infrabinv1connect.NewInfrabinServiceHandler(infraServer)
	mux.Handle(path, handler)
	http.ListenAndServe(
		"localhost:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
