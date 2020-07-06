package infrabin

import "testing"

func TestNewGRPCServer(t *testing.T) {
	config := DefaultConfig()
	server := NewGRPCServer(config)
	if server.Name != "grpc" {
		t.Errorf("Name not set on GRPCServer. got %v want %v", server.Name, "grpc")
	}
	if server.Config != config {
		t.Errorf("Config not set on GRPCServer: %v", server)
	}
}
