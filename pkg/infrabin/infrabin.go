package infrabin

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/maruina/go-infrabin/internal/aws"
	"github.com/maruina/go-infrabin/internal/helpers"
	"github.com/spf13/viper"
)

// Must embed UnimplementedInfrabinServer for `protogen-gen-go-grpc`
type InfrabinService struct {
	UnimplementedInfrabinServer
	STSClient aws.STSApi
}

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
	maxDelay := viper.GetDuration("maxDelay")
	requestDuration := time.Duration(request.Duration) * time.Second

	duration := helpers.MinDuration(requestDuration, maxDelay)
	time.Sleep(duration)

	return &Response{Delay: int32(duration.Seconds())}, nil
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
	if !viper.GetBool("proxyEndpoint") {
		return nil, status.Errorf(codes.Unimplemented, "Proxy endpoint disabled. Enabled with --enable-proxy-endpoint")
	}
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
	// Error checks
	if request.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Path must not be empty")
	}

	u, err := url.Parse(viper.GetString("awsMetadataEndpoint"))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "s.Config.AWSMetadataEndpoint invalid: %v", err)
	}

	// If calling the metadata endpoint
	if strings.HasPrefix(request.Path, "metadata") {
		u.Path = request.Path
		return s.Proxy(ctx, &ProxyRequest{Method: "GET", Url: u.String()})
	}

	// If calling to assume a role
	if strings.HasPrefix(request.Path, "assume") {
		roleArn := strings.TrimPrefix(request.Path, "assume/")
		fmt.Printf("roleArn is: %v\n", roleArn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error creating AWS client, %v", err)
		}
		roleId, err := aws.STSAssumeRole(ctx, s.STSClient, roleArn, "aws-assume-session-go-infrabin")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error assuming AWS IAM role, %v", err)
		}
		responseMap, err := structpb.NewValue(map[string]interface{}{
			"assumedRoleId": roleId,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error creating reponse with AWS AssumedRoleId %v, %v", roleId, err)
		}
		return structpb.NewStruct(responseMap.GetStructValue().AsMap())
	}

	return nil, status.Errorf(codes.InvalidArgument, "Supported paths are /aws/metadata or /aws/assume, got %v", request.Path)
}

func (s *InfrabinService) Any(ctx context.Context, request *AnyRequest) (*Response, error) {
	return &Response{Path: request.Path}, nil
}
