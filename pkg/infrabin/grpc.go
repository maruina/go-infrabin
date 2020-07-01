package infrabin

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server and implements infrabin.Infrabin
type GRPCServer struct {
	Name   string
	Server *grpc.Server
}

// ListenAndServe binds the server to the indicated interface:port.
func (s *GRPCServer) ListenAndServe() {
	ln, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Listen failed on 0.0.0.0:50051: %v", err)
	}

	log.Printf("Starting %s server on %s", s.Name, ln.Addr())
	if err := s.Server.Serve(ln); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *GRPCServer) Shutdown() {
	log.Printf("Shutting down %s server with GracefulStop()", s.Name)
	s.Server.GracefulStop()
	log.Printf("GRPC %s server stopped", s.Name)
}

// New creates a new rpc server.
func NewGRPCServer() *GRPCServer {
	gs := grpc.NewServer()
	s := &GRPCServer{Name: "grpc", Server: gs}
	is := &InfrabinService{}
	RegisterInfrabinServer(gs, is)
	reflection.Register(gs)
	return s
}
