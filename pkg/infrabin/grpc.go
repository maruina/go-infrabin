package infrabin

import (
	"context"
	"fmt"
	"log"
	"net"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/maruina/go-infrabin/internal/aws"
	"github.com/maruina/go-infrabin/internal/k8s"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// k8sClientWrapper wraps the internal k8s.ClientAdapter and implements the K8sClient interface.
// It converts between internal k8s.PodInfo and infrabin.K8sPodInfo types.
type k8sClientWrapper struct {
	adapter *k8s.ClientAdapter
}

// DiscoverPods implements the K8sClient interface by converting k8s.PodInfo to K8sPodInfo.
func (w *k8sClientWrapper) DiscoverPods(ctx context.Context, labelSelector string) ([]K8sPodInfo, error) {
	pods, err := w.adapter.GetPods(ctx, labelSelector)
	if err != nil {
		return nil, err
	}

	// Convert internal PodInfo to K8sPodInfo
	result := make([]K8sPodInfo, len(pods))
	for i, pod := range pods {
		result[i] = K8sPodInfo{
			Name:             pod.Name,
			IP:               pod.IP,
			AvailabilityZone: pod.AvailabilityZone,
		}
	}

	return result, nil
}

// GRPCServer wraps the gRPC server and implements infrabin.Infrabin
type GRPCServer struct {
	Name            string
	Server          *grpc.Server
	InfrabinService InfrabinServer
	HealthService   *health.Server
}

// ListenAndServe binds the server to the indicated interface:port.
// Pass lis as nil to be bound to the config port. lis can be passed for testing
func (s *GRPCServer) ListenAndServe(lis net.Listener) {
	addr := viper.GetString(s.Name+".host") + ":" + viper.GetString(s.Name+".port")

	var err error
	if lis == nil {
		if lis, err = net.Listen("tcp", addr); err != nil {
			log.Printf("ERROR: Listen failed on %s: %v", addr, err)
			return
		}
	}

	log.Printf("Starting %s server on %s", s.Name, lis.Addr())
	if err = s.Server.Serve(lis); err != nil {
		log.Printf("ERROR: gRPC %s server failed: %v", s.Name, err)
	}
}

func (s *GRPCServer) Shutdown() {
	drainTimeout := viper.GetDuration("drainTimeout")
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	gracefulDone := make(chan struct{}, 1)

	go func() {
		log.Printf("Set all serving status to NOT_SERVING")
		s.HealthService.Shutdown()
		log.Printf("Shutting down %s server with %s GracefulStop()", s.Name, drainTimeout)
		s.Server.GracefulStop()
		log.Printf("gRPC %s server stopped", s.Name)
		gracefulDone <- struct{}{}
	}()

	select {
	case <-gracefulDone:
		return
	case <-ctx.Done():
		log.Printf("Shutting down %s server with Stop() as it took too long", s.Name)
		s.Server.Stop()
		return
	}
}

// NewGRPCServer creates a new gRPC server.
// Returns an error if server initialization fails.
func NewGRPCServer() (*GRPCServer, error) {
	gs := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)

	// Create the gRPC services
	healthServer := health.NewServer()
	stsClient, err := aws.GetSTSClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS STS client: %w", err)
	}

	// Initialize Kubernetes client if crossaz endpoint is enabled
	var k8sClient K8sClient
	if viper.GetBool("enableCrossAZEndpoint") {
		client, err := k8s.NewInClusterClient()
		if err != nil {
			// If the endpoint is explicitly enabled but we can't initialize the client,
			// fail fast rather than silently running with a broken endpoint.
			return nil, fmt.Errorf("failed to initialize Kubernetes client for CrossAZ endpoint: %w", err)
		}
		adapter := k8s.NewClientAdapter(client)
		k8sClient = &k8sClientWrapper{adapter: adapter}
		log.Printf("Kubernetes client initialized for CrossAZ endpoint")
	}

	infrabinService := &InfrabinService{
		STSClient:     stsClient,
		HealthService: healthServer,
		K8sClient:     k8sClient,
	}

	// Register gRPC services on the grpc server
	RegisterInfrabinServer(gs, infrabinService)
	grpc_health_v1.RegisterHealthServer(gs, healthServer)
	grpc_prometheus.Register(gs)
	reflection.Register(gs)

	// Set the health of the infrabin service to healthy
	healthServer.SetServingStatus("infrabin.Infrabin", grpc_health_v1.HealthCheckResponse_SERVING)
	// Set liveness and readiness to healthy by default
	healthServer.SetServingStatus("liveness", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("readiness", grpc_health_v1.HealthCheckResponse_SERVING)

	return &GRPCServer{
		Name:            "grpc",
		Server:          gs,
		InfrabinService: infrabinService,
		HealthService:   healthServer,
	}, nil
}
