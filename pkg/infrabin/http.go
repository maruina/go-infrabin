package infrabin

import (
	"context"
	"fmt"
	"github.com/maruina/go-infrabin/internal/helpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"log"
	"net/http"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

type GRPCHandlerFunc func(context.Context, interface{}) (*Response, error)
type RequestBuilder func(*http.Request) (interface{}, error)

type Error struct {
	e *spb.Status
}
func (e Error) Error() string {
	return fmt.Sprintf("rpc error: code = %s desc = %s", codes.Code(e.e.GetCode()), e.e.GetMessage())
}

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
