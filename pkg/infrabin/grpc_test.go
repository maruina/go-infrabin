package infrabin

import "testing"

func TestNewGRPCServer(t *testing.T) {

	server := NewGRPCServer()
	if server.Name != "grpc" {
		t.Errorf("Name not set on GRPCServer. got %v want %v", server.Name, "grpc")
	}
}
