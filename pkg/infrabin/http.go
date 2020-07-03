package infrabin

import (
	"context"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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

func NewHTTPServer(config *Config) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux(runtime.WithLastMatchWins())
	is := InfrabinService{}
	err := RegisterInfrabinHandlerServer(ctx, mux, &is)
	if err != nil {
		log.Fatalf("gRPC %s server failed to register: %v", err)
	}

	server := &http.Server{
		Handler: mux,
		Addr:    "0.0.0.0:8888",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return &HTTPServer{Name: "service", Config: config, Server: server}
}

func NewAdminServer(config *Config) *HTTPServer {
	r := mux.NewRouter()

	r.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Name("Liveness")

	server := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8899",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &HTTPServer{Name: "admin", Config: config, Server: server}
}
