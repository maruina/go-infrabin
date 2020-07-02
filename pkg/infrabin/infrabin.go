package infrabin

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/maruina/go-infrabin/internal/helpers"
)

const AWS_METADATA_ENDPOINT = "http://169.254.169.254/latest/meta-data/"

type InfrabinService struct{}

func (s *InfrabinService) Root(ctx context.Context, _ *Empty) (*Response, error) {
	fail := helpers.GetEnv("FAIL_ROOT_HANDLER", "")
	if fail != "" {
		return nil, status.Error(codes.Unavailable, "some description")
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("cannot get hostname: %v", err)
		}

		var resp Response
		resp.Hostname = hostname
		resp.Kubernetes = &KubeResponse{
			PodName:   helpers.GetEnv("POD_NAME", ""),
			Namespace: helpers.GetEnv("POD_NAMESPACE", ""),
			PodIp:     helpers.GetEnv("POD_IP", ""),
			NodeName:  helpers.GetEnv("NODE_NAME", ""),
		}
		return &resp, nil
	}
}

func (s *InfrabinService) Delay(ctx context.Context, request *DelayRequest) (*Response, error) {
	maxDelay, err := strconv.Atoi(helpers.GetEnv("INFRABIN_MAX_DELAY", "120"))
	if err != nil {
		log.Fatalf("cannot convert env var INFRABIN_MAX_DELAY to integer: %v", err)
		return nil, status.Error(codes.Internal, "cannot convert env var INFRABIN_MAX_DELAY to integer")
	}

	seconds := helpers.Min(int(request.Duration), maxDelay)
	time.Sleep(time.Duration(seconds) * time.Second)

	return &Response{Delay: int32(seconds)}, nil
}

func (s *InfrabinService) Liveness(ctx context.Context, _ *Empty) (*Response, error) {
	return &Response{Liveness: "pass"}, nil
}

func (s *InfrabinService) Env(ctx context.Context, request *EnvRequest) (*Response, error) {
	value := helpers.GetEnv(request.EnvVar, "")
	if value == "" {
		return nil, status.Errorf(codes.NotFound, "No env var named %s", request.EnvVar)
	} else {
		return &Response{Env: map[string]string{request.EnvVar: value}}, nil
	}
}

func (s *InfrabinService) Headers(ctx context.Context, request *HeadersRequest) (*Response, error) {
	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}
	md, _ := metadata.FromIncomingContext(ctx)
	for key := range md {
		request.Headers[key] = strings.Join(md.Get(key), ",")
	}
	return &Response{Headers: request.Headers}, nil
}

func (s *InfrabinService) Proxy(ctx context.Context, request *ProxyRequest) (*structpb.Struct, error) {
	// Convert Struct into json []byte
	requestBody, err := request.Body.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to marshal downstream request body: %v", err)
	}

	// Make upstream request from incoming request
	req, err := http.NewRequestWithContext(ctx, request.Method, request.Url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to build request: %v", err)
	}
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// Send http request
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach %s: %v", request.Url, err)
	}

	// Read request body and close it
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error reading upstream response body: %v", err)
	}
	if err = resp.Body.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "Error closing upstream response: %v", err)
	}

	// Convert []bytes into json struct
	var response structpb.Struct
	if err := response.UnmarshalJSON(body); err != nil {
		return nil, status.Errorf(codes.Internal, "Error creating Struct from upstream response json: %v", err)
	}
	return &response, nil
}

func (s *InfrabinService) AWS(ctx context.Context, request *AWSRequest) (*structpb.Struct, error) {
	if request.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Path must not be empty")
	}
	return s.Proxy(ctx, &ProxyRequest{Method: "GET", Url: AWS_METADATA_ENDPOINT + request.Path})
}
