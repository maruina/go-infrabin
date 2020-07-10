package infrabin

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/maruina/go-infrabin/pkg/grpc/health/v1"
	"google.golang.org/grpc/health"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPServerOption func(ctx context.Context, s *HTTPServer)

func RegisterHealth(healthService *health.Server) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) {
		// Register the handler to call local instance, i.e. no network calls
		err := grpc_health_v1.RegisterHealthHandlerServer(
			ctx,
			s.Server.Handler.(*runtime.ServeMux),
			healthService,
		)
		if err != nil {
			log.Fatalf("gRPC server failed to register: %v", err)
		}
	}
}

func RegisterInfrabin(infrabinService InfrabinServer) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) {
		// Register the handler to call local instance, i.e. no network calls
		err := RegisterInfrabinHandlerServer(
			ctx,
			s.Server.Handler.(*runtime.ServeMux),
			infrabinService,
		)
		if err != nil {
			log.Fatalf("gRPC server failed to register: %v", err)
		}
	}
}

type HTTPServer struct {
	Name   string
	Server *http.Server
}

func (s *HTTPServer) ListenAndServe() {
	log.Printf("Starting %s server on %s", s.Name, s.Server.Addr)
	if err := s.Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("HTTP server crashed", err)
	}
}

func (s *HTTPServer) Shutdown() {
	log.Printf("Shutting down %s server with 15s graceful shutdown", s.Name)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP %s server graceful shutdown failed: %v", s.Name, err)
	} else {
		log.Printf("HTTP %s server stopped", s.Name)
	}
}

func NewHTTPServer(name string, addr string, opts ...HTTPServerOption) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(passThroughHeaderMatcher),
	)

	// Set default marshaller options
	marshaler, _ := runtime.MarshalerForRequest(mux, &http.Request{})
	jsonMarshaler := marshaler.(*runtime.HTTPBodyMarshaler).Marshaler.(*runtime.JSONPb)
	jsonMarshaler.EmitUnpopulated = false
	jsonMarshaler.UseProtoNames = true

	// A standard http.Server
	server := &http.Server{
		Handler: mux,
		Addr:    addr,
		// Good practice: enforce timeouts
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	s := &HTTPServer{Name: name, Server: server}
	for _, opt := range opts {
		opt(ctx, s)
	}
	return s
}

// NewPromServer creates a new HTTP server with a Prometheus handler
func NewPromServer(name string, addr string, config *Config) *HTTPServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	promServer := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Good practice: enforce timeouts
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return &HTTPServer{Name: name, Server: promServer}
}

// Keep the standard "Grpc-Metadata-" and well known behaviour
// All other headers are passed, also with grpcgateway- prefix
func passThroughHeaderMatcher(key string) (string, bool) {
	if grpcKey, ok := runtime.DefaultHeaderMatcher(key); ok {
		return grpcKey, ok
	} else {
		return runtime.MetadataPrefix + key, true
	}
}

// Workaround for not being able to specify root as a path
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/1500
func init() {
	pattern_Infrabin_Root_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0}, []string{""}, ""))
}
