package infrabin

import (
	"context"
	"log"
	"net"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server and implements infrabin.Infrabin
type GRPCServer struct {
	Name            string
	Server          *grpc.Server
	InfrabinService InfrabinServer
	HealthService   *health.Server
}

// ListenAndServe binds the server to the indicated interface:port.
func (s *GRPCServer) ListenAndServe() {
	addr := viper.GetString(s.Name+".host") + ":" + viper.GetString(s.Name+".port")

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Listen failed on "+addr+": %v", err)
	}

	log.Printf("Starting %s server on %s", s.Name, ln.Addr())
	if err := s.Server.Serve(ln); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *GRPCServer) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	gracefulDone := make(chan struct{}, 1)

	go func() {
		log.Printf("Set all serving status to NOT_SERVING")
		s.HealthService.Shutdown()
		log.Printf("Shutting down %s server with GracefulStop()", s.Name)
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

// New creates a new rpc server.
func NewGRPCServer() *GRPCServer {
	gs := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)

	// Create the gPRC services
	healthServer := health.NewServer()
	infrabinService := &InfrabinService{
		LivenessHealthService:  healthServer,
		ReadinessHealthService: healthServer,
	}

	// Register gRPC services on the grpc server
	RegisterInfrabinServer(gs, infrabinService)
	grpc_health_v1.RegisterHealthServer(gs, healthServer)
	grpc_prometheus.Register(gs)
	reflection.Register(gs)

	// Set the health of the infrabin service to healthy
	healthServer.SetServingStatus("infrabin.Infrabin", grpc_health_v1.HealthCheckResponse_SERVING)

	return &GRPCServer{
		Name:            "grpc",
		Server:          gs,
		InfrabinService: infrabinService,
		HealthService:   healthServer,
	}
}
