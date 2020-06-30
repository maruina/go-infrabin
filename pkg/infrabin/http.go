package infrabin

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/maruina/go-infrabin/internal/helpers"
)

type RequestBuilder func(*http.Request) (interface{}, error)

//func(context.Content, interface{}) (interface{}, error)
func MakeHandler(grpcHandler grpc.UnaryHandler, requestBuilder RequestBuilder) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		request, err := requestBuilder(r)
		if err != nil {
			log.Fatalf("Failed to build request: %v", err)
		}

		resp, err := grpcHandler(context.Background(), request)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
		data, _ := marshalOptions.Marshal(resp.(protoreflect.ProtoMessage))
		_, err = io.WriteString(w, string(data))
		if err != nil {
			log.Fatal("error writing to ResponseWriter: ", err)
		}
	}
}

// LivenessHandler handles the "/liveness" endpoint
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp.Liveness = "pass"
	data := helpers.MarshalResponseToString(resp)
	_, err := io.WriteString(w, data)
	if err != nil {
		log.Fatal("error writing to ResponseWriter", err)
	}
}

// DelayHandler handles the "/delay" endpoint
func DelayHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	var seconds int
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	seconds, err := strconv.Atoi(vars["seconds"])
	if err != nil {
		resp.Error = "cannot convert seconds to integer"
		data := helpers.MarshalResponseToString(resp)
		w.WriteHeader(http.StatusBadRequest)
		_, err = io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	} else {
		maxDelay, err := strconv.Atoi(helpers.GetEnv("INFRABIN_MAX_DELAY", "120"))
		if err != nil {
			log.Fatalf("cannot convert env var INFRABIN_MAX_DELAY to integer: %v", err)
		}
		time.Sleep(time.Duration(helpers.Min(seconds, maxDelay)) * time.Second)

		resp.Delay = strconv.Itoa(seconds)
		data := helpers.MarshalResponseToString(resp)

		w.WriteHeader(http.StatusOK)
		_, err = io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	}
}

// HeadersHandler handles the headers endpoint
func HeadersHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	w.Header().Set("Content-Type", "application/json")
	resp.Headers = &r.Header

	data := helpers.MarshalResponseToString(resp)
	_, err := io.WriteString(w, data)
	if err != nil {
		log.Fatal("error writing to ResponseWriter: ", err)
	}
}

// EnvHandler handles the env endpoint
func EnvHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	value := helpers.GetEnv(vars["env_var"], "")

	if value == "" {
		data := helpers.MarshalResponseToString(resp)
		w.WriteHeader(http.StatusNotFound)
		_, err := io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	} else {

		resp.Env = map[string]string{
			vars["env_var"]: value,
		}
		data := helpers.MarshalResponseToString(resp)

		w.WriteHeader(http.StatusOK)
		_, err := io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}

	}
}
