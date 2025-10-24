package infrabin

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/spf13/viper"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type HTTPServer struct {
	Name   string
	Server *http.Server
}

type HTTPServerOption func(ctx context.Context, s *HTTPServer)

func RegisterInfrabin(pattern string, infrabinService InfrabinServer) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) {
		// Register the handler to call local instance, i.e. no network calls
		mux := newGatewayMux()
		if err := RegisterInfrabinHandlerServer(ctx, mux, infrabinService); err != nil {
			log.Fatalf("gRPC server failed to register: %v", err)
		}

		// Wrap with metrics middleware
		instrumentedHandler := HTTPMetricsMiddleware(mux)

		var handler http.Handler
		if p := strings.TrimSuffix(pattern, "/"); len(p) < len(pattern) {
			handler = http.StripPrefix(p, instrumentedHandler)
		} else {
			handler = instrumentedHandler
		}
		s.Server.Handler.(*http.ServeMux).Handle(pattern, handler)
	}
}

func RegisterHandler(pattern string, handler http.Handler) HTTPServerOption {
	return func(ctx context.Context, s *HTTPServer) {
		if p := strings.TrimSuffix(pattern, "/"); len(p) < len(pattern) {
			handler = http.StripPrefix(p, handler)
		}
		s.Server.Handler.(*http.ServeMux).Handle(pattern, handler)
	}
}

func (s *HTTPServer) ListenAndServe() {
	// Wrap handler now that everything is registered
	s.Server.Handler = handlers.RecoveryHandler()(handlers.ProxyHeaders(handlers.CombinedLoggingHandler(os.Stdout, s.Server.Handler)))

	log.Printf("Starting %s server on %s", s.Name, s.Server.Addr)
	if err := s.Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("HTTP server crashed", err)
	}
}

func (s *HTTPServer) Shutdown() {
	drainTimeout := viper.GetDuration("drainTimeout")
	log.Printf("Shutting down %s server with %s graceful shutdown", s.Name, drainTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP %s server graceful shutdown failed: %v", s.Name, err)
	} else {
		log.Printf("HTTP %s server stopped", s.Name)
	}
}

func NewHTTPServer(name string, opts ...HTTPServerOption) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		opt(ctx, s)
	}
	return s
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
	} else {
		return runtime.MetadataPrefix + key, true
	}
}

// Workaround for not being able to specify root as a path
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/1500
func init() {
	pattern_Infrabin_Root_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0}, []string{""}, ""))
}
