package infrabin

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/gorilla/mux"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func BuildEmpty(r *http.Request) (proto.Message, error) {
	return &Empty{}, nil
}

func BuildDelayRequest(request *http.Request) (proto.Message, error) {
	vars := mux.Vars(request)
	if seconds, err := strconv.Atoi(vars["seconds"]); err != nil {
		return nil, err
	} else {
		return &DelayRequest{Duration: int32(seconds)}, nil
	}
}

func BuildEnvRequest(request *http.Request) (proto.Message, error) {
	vars := mux.Vars(request)
	return &EnvRequest{EnvVar: vars["env_var"]}, nil
}

func BuildHeadersRequest(request *http.Request) (proto.Message, error) {
	inputHeaders := textproto.MIMEHeader(request.Header)
	headers := make(map[string]string)
	for key := range inputHeaders {
		headers[key] = inputHeaders.Get(key)
	}
	return &HeadersRequest{Headers: headers}, nil
}

func BuildProxyRequest(request *http.Request) (proto.Message, error) {
	// Read request body
	inputBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body: %v", err)
	}

	// Create Struct
	var body structpb.Struct
	if err := body.UnmarshalJSON(inputBody); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal body json: %v", err)
	}

	// Make map[string]string headers
	headers := make(map[string]string)
	for key, value := range body.Fields["headers"].GetStructValue().AsMap() {
		headers[key] = fmt.Sprintf("%s", value)
	}

	proxyRequest := &ProxyRequest{
		Method:  body.Fields["method"].GetStringValue(),
		Url:     body.Fields["url"].GetStringValue(),
		Body:    body.Fields["body"].GetStructValue(),
		Headers: headers,
	}
	return proxyRequest, nil
}

func BuildAWSRequest(request *http.Request) (proto.Message, error) {
	vars := mux.Vars(request)
	return &AWSRequest{Path: vars["path"]}, nil
}
