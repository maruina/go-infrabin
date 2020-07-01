package infrabin

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"log"
	"net/http"
)

type GRPCHandlerFunc func(ctx context.Context, request interface{}) (*Response, error)
type RequestBuilder func(*http.Request) (interface{}, error)


//func(context.Content, interface{}) (interface{}, error)
func MakeHandler(grpcHandler GRPCHandlerFunc, requestBuilder RequestBuilder) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var (
			response *Response
			err error
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

