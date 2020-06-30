package infrabin

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

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

func NewHTTPServer() *HTTPServer {
	r := mux.NewRouter()
	is := InfrabinService{}

	r.HandleFunc("/", MakeHandler(
		func(ctx context.Context, req interface{}) (*Response, error) {
			return is.Root(ctx, req.(*Empty))
		},
		func(r *http.Request) (interface{}, error) {
			return &Empty{}, nil
		},
	)).Name("Root")

	r.HandleFunc("/delay/{seconds}", MakeHandler(
		func(ctx context.Context, req interface{}) (*Response, error) {
			return is.Delay(ctx, req.(*DelayRequest))
		},
		func(request *http.Request) (i interface{}, e error) {
			vars := mux.Vars(request)
			if seconds, err := strconv.Atoi(vars["seconds"]); err != nil {
				return nil, err
			} else {
				return &DelayRequest{Duration: int32(seconds)}, nil
			}
		},
	)).Name("Delay")

	r.HandleFunc("/env/{env_var}", MakeHandler(
		func(ctx context.Context, req interface{}) (*Response, error) {
			return is.Env(ctx, req.(*EnvRequest))
		},
		func(request *http.Request) (i interface{}, e error) {
			vars := mux.Vars(request)
			return &EnvRequest{EnvVar: vars["env_var"]}, nil
		},
	)).Name("Env")

	r.HandleFunc("/headers", HeadersHandler)

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
	is := InfrabinService{}

	r.HandleFunc("/liveness", MakeHandler(
		func(ctx context.Context, req interface{}) (*Response, error) {
			return is.Liveness(ctx, req.(*Empty))
		},
		func(request *http.Request) (i interface{}, e error) {
			return &Empty{}, nil
		},
	)).Name("Liveness")

	server := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8899",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &HTTPServer{Name: "admin", Server: server}
}
