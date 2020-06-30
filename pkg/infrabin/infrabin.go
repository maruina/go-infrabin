package infrabin

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maruina/go-infrabin/internal/helpers"
)

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
