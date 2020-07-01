package infrabin

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
)

type GRPCHandlerFunc func(ctx context.Context, request interface{}) (*Response, error)
type RequestBuilder func(*http.Request) (interface{}, error)

//func(context.Content, interface{}) (interface{}, error)
func MakeHandler(grpcHandler GRPCHandlerFunc, requestBuilder RequestBuilder) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var (
			response *Response
			err      error
		)
		request, err := requestBuilder(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response = &Response{Error: fmt.Sprintf("Failed to build request: %v", err)}
		} else {
			response, err = grpcHandler(r.Context(), request)
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}

		marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
		data, _ := marshalOptions.Marshal(response)
		_, err = io.WriteString(w, string(data))
		if err != nil {
			log.Fatal("error writing to ResponseWriter: ", err)
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

	r.HandleFunc("/headers", MakeHandler(
		func(ctx context.Context, request interface{}) (*Response, error) {
			return is.Headers(ctx, request.(*HeadersRequest))
		},
		func(request *http.Request) (i interface{}, e error) {
			inputHeaders := textproto.MIMEHeader(request.Header)
			headers := make(map[string]string)
			for key := range inputHeaders {
				headers[key] = inputHeaders.Get(key)
			}
			return &HeadersRequest{Headers: headers}, nil
		},
	)).Name("Headers")

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
