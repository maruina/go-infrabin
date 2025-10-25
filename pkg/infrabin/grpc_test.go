package infrabin

import (
	"testing"
)

func TestNewGRPCServer(t *testing.T) {
	server, err := NewGRPCServer()
	if err != nil {
		t.Fatalf("NewGRPCServer() returned error: %v", err)
	}
	if server.Name != "grpc" {
		t.Errorf("Name not set on GRPCServer. got %v want %v", server.Name, "grpc")
	}
}
