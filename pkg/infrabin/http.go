package infrabin

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

type HTTPServer struct {
	Name   string
	Config *Config
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

func NewHTTPServer(name string, addr string, config *Config) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux()
	is := InfrabinService{}
	err := RegisterInfrabinHandlerServer(ctx, mux, &is)
	if err != nil {
		log.Fatalf("gRPC server failed to register: %v", err)
	}

	server := &http.Server{
		Handler: mux,
		Addr: addr,
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return &HTTPServer{Name: name, Config: config, Server: server}
}

func init() {
	pattern_Infrabin_Root_0 = runtime.MustPattern(
		runtime.NewPattern(1, []int{2, 0}, []string{""}, "", runtime.AssumeColonVerbOpt(true)),
	)
}
