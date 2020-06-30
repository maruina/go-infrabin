package infrabin

import (
	"context"
	"log"
	"os"

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
