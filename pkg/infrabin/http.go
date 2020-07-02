package infrabin

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCHandlerFunc func(ctx context.Context, request interface{}) (proto.Message, error)
type RequestBuilder func(*http.Request) (proto.Message, error)

//func(context.Content, interface{}) (interface{}, error)
func MakeHandler(grpcHandler GRPCHandlerFunc, requestBuilder RequestBuilder) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var (
			response proto.Message
			err      error
		)
		request, err := requestBuilder(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response = &Response{Error: fmt.Sprintf("Failed to build request: %v", err)}
		} else {
			response, err = grpcHandler(r.Context(), request)
			if err != nil {
				log.Printf("Error from grpcHandler: %v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				// If the response is of Type Response we can add Error
				if resp, ok := response.(*Response); ok {
					if resp == nil {
						resp = &Response{}
					}
					resp.Error = fmt.Sprintf("Error from grpcHandler: %v", err)
					response = resp
				}
				// If the response is of Type Struct we can add Error
				if resp, ok := response.(*structpb.Struct); ok {
					if resp == nil {
						resp = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
					}
					resp.Fields["error"] = structpb.NewStringValue(
						fmt.Sprintf("Error from grpcHandler: %v", err),
					)
					response = resp
				}
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
		func(ctx context.Context, req interface{}) (proto.Message, error) {
			return is.Root(ctx, req.(*Empty))
		},
		BuildEmpty,
	)).Name("Root")

	r.HandleFunc("/delay/{seconds}", MakeHandler(
		func(ctx context.Context, req interface{}) (proto.Message, error) {
			return is.Delay(ctx, req.(*DelayRequest))
		},
		BuildDelayRequest,
	)).Name("Delay")

	r.HandleFunc("/env/{env_var}", MakeHandler(
		func(ctx context.Context, req interface{}) (proto.Message, error) {
			return is.Env(ctx, req.(*EnvRequest))
		},
		BuildEnvRequest,
	)).Name("Env")

	r.HandleFunc("/headers", MakeHandler(
		func(ctx context.Context, request interface{}) (proto.Message, error) {
			return is.Headers(ctx, request.(*HeadersRequest))
		},
		BuildHeadersRequest,
	)).Name("Headers")

	r.HandleFunc("/proxy", MakeHandler(
		func(ctx context.Context, request interface{}) (proto.Message, error) {
			return is.Proxy(ctx, request.(*ProxyRequest))
		},
		BuildProxyRequest,
	)).Methods("POST").Name("Proxy")

	r.HandleFunc("/aws/{path}", MakeHandler(
		func(ctx context.Context, request interface{}) (proto.Message, error) {
			return is.AWS(ctx, request.(*AWSRequest))
		},
		BuildAWSRequest,
	)).Name("AWS")

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
		func(ctx context.Context, req interface{}) (proto.Message, error) {
			return is.Liveness(ctx, req.(*Empty))
		},
		BuildEmpty,
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
