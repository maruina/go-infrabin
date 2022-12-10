package server

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/connect-go"
	infrabinv1 "github.com/maruina/go-infrabin/gen/infrabin/v1" // generated by protoc-gen-go
)

type InfrabinServer struct{}

func (s *InfrabinServer) Headers(ctx context.Context, req *connect.Request[infrabinv1.HeadersRequest]) (*connect.Response[infrabinv1.HeadersResponse], error) {
	res := connect.NewResponse(&infrabinv1.HeadersResponse{
		Headers: map[string]string{},
	})
	for k, vv := range req.Header() {
		for _, v := range vv {
			res.Msg.Headers[k] = v
		}
	}
	return res, nil
}

func (s *InfrabinServer) Env(ctx context.Context, req *connect.Request[infrabinv1.EnvRequest]) (*connect.Response[infrabinv1.EnvResponse], error) {
	res := connect.NewResponse(&infrabinv1.EnvResponse{
		Environment: map[string]string{},
	})
	if req.Msg.Key == "" {
		for _, item := range os.Environ() {
			a := strings.Split(item, "=")
			// Filter our any environment variable which is not "key=value"
			if len(a) == 2 {
				res.Msg.Environment[a[0]] = a[1]
			}
		}
		return res, nil
	}

	value, ok := os.LookupEnv(req.Msg.Key)
	if !ok {
		// If the environment variable does no exist return 404
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	res.Msg.Environment[req.Msg.Key] = value
	return res, nil
}

func (s *InfrabinServer) Root(ctx context.Context, req *connect.Request[infrabinv1.RootRequest]) (*connect.Response[infrabinv1.RootResponse], error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&infrabinv1.RootResponse{
		Hostname: hostname,
	})

	return res, nil
}

func (s *InfrabinServer) Delay(ctx context.Context, req *connect.Request[infrabinv1.DelayRequest]) (*connect.Response[infrabinv1.DelayResponse], error) {
	time.Sleep(req.Msg.Duration.AsDuration())

	res := connect.NewResponse(&infrabinv1.DelayResponse{})

	return res, nil
}
