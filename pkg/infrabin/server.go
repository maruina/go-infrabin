package infrabin

import (
	"log"
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type HTTPServer struct {
	Name string
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
	ctx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP %s server graceful shutdown failed: %v", s.Name, err)
	} else {
		log.Printf("HTTP %s server stopped", s.Name)
	}
}

func NewHTTPServer() *HTTPServer {
	r := mux.NewRouter()

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/delay/{seconds}", DelayHandler)
	r.HandleFunc("/headers", HeadersHandler)
	r.HandleFunc("/env/{env_var}", EnvHandler)

	server := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8888",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return &HTTPServer{Name: "service", Server: server}
}

func NewAdminServer() *HTTPServer {
	r := mux.NewRouter()

	r.HandleFunc("/liveness", LivenessHandler)

	server := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8899",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &HTTPServer{Name: "admin", Server: server}
}

