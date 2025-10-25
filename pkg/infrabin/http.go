package infrabin

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/spf13/viper"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

//go:embed openapi.swagger.json
var openAPISpec []byte

type HTTPServer struct {
	Name   string
	Server *http.Server
}

type HTTPServerOption func(ctx context.Context, s *HTTPServer) error

func RegisterInfrabin(pattern string, infrabinService InfrabinServer) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) error {
		// Register the handler to call local instance, i.e. no network calls
		gatewayMux := newGatewayMux()
		if err := RegisterInfrabinHandlerServer(ctx, gatewayMux, infrabinService); err != nil {
			return fmt.Errorf("failed to register infrabin handler: %w", err)
		}

		// Wrap with metrics middleware
		handler := HTTPMetricsMiddleware(gatewayMux)

		// Register the handler on the HTTP server's ServeMux
		serveMux, ok := s.Server.Handler.(*http.ServeMux)
		if !ok {
			return fmt.Errorf("handler is not *http.ServeMux")
		}
		serveMux.Handle(pattern, handler)
		return nil
	}
}

func RegisterHandler(pattern string, handler http.Handler) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) error {
		serveMux, ok := s.Server.Handler.(*http.ServeMux)
		if !ok {
			return fmt.Errorf("handler is not *http.ServeMux")
		}
		serveMux.Handle(pattern, handler)
		return nil
	}
}

// RegisterOpenAPI registers an OpenAPI specification handler at /openapi.json.
// This serves the embedded OpenAPI spec generated from the protobuf definitions.
func RegisterOpenAPI(pattern string) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) error {
		serveMux, ok := s.Server.Handler.(*http.ServeMux)
		if !ok {
			return fmt.Errorf("handler is not *http.ServeMux")
		}
		serveMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(openAPISpec)
		})
		return nil
	}
}

func (s *HTTPServer) ListenAndServe() {
	// Wrap handler now that everything is registered
	handler := handlers.CustomLoggingHandler(os.Stdout, s.Server.Handler, RequestLoggingFormatter)
	handler = handlers.ProxyHeaders(handler)
	handler = handlers.RecoveryHandler()(handler)

	s.Server.Handler = handler

	log.Printf("Starting %s server on %s", s.Name, s.Server.Addr)
	if err := s.Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("ERROR: HTTP %s server crashed: %v", s.Name, err)
	}
}

func (s *HTTPServer) Shutdown() {
	drainTimeout := viper.GetDuration("drainTimeout")
	log.Printf("Shutting down %s server with %s graceful shutdown", s.Name, drainTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		log.Printf("ERROR: HTTP %s server graceful shutdown failed: %v", s.Name, err)
		return
	}
	log.Printf("HTTP %s server stopped", s.Name)
}

func NewHTTPServer(name string, opts ...HTTPServerOption) (*HTTPServer, error) {
	// Use Background context for handler registration
	// This context is only used during initialization, not for runtime cancellation
	ctx := context.Background()

	addr := viper.GetString(name+".host") + ":" + viper.GetString(name+".port")

	// A standard http.Server
	server := &http.Server{
		Handler: http.NewServeMux(),
		Addr:    addr,
		// Good practice: enforce timeouts
		WriteTimeout:      viper.GetDuration("httpWriteTimeout"),
		ReadTimeout:       viper.GetDuration("httpReadTimeout"),
		IdleTimeout:       viper.GetDuration("httpIdleTimeout"),
		ReadHeaderTimeout: viper.GetDuration("httpReadHeaderTimeout"),
	}

	s := &HTTPServer{Name: name, Server: server}
	for _, opt := range opts {
		if err := opt(ctx, s); err != nil {
			return nil, fmt.Errorf("failed to apply HTTP server option: %w", err)
		}
	}
	return s, nil
}

func newGatewayMux() *runtime.ServeMux {
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(passThroughHeaderMatcher),
	)
	// Set default marshaller options
	marshaler, _ := runtime.MarshalerForRequest(mux, &http.Request{})
	jsonMarshaler := marshaler.(*runtime.HTTPBodyMarshaler).Marshaler.(*runtime.JSONPb)
	jsonMarshaler.EmitUnpopulated = false
	jsonMarshaler.UseProtoNames = true
	return mux
}

// Keep the standard "Grpc-Metadata-" and well known behaviour
// All other headers are passed, also with grpcgateway- prefix
func passThroughHeaderMatcher(key string) (string, bool) {
	if grpcKey, ok := runtime.DefaultHeaderMatcher(key); ok {
		return grpcKey, ok
	}
	return runtime.MetadataPrefix + key, true
}

// Workaround for not being able to specify root as a path
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/1500
func init() {
	pattern_Infrabin_Root_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0}, []string{""}, ""))
}
