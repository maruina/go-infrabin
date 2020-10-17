package infrabin

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewGRPCServer(t *testing.T) {

	server := NewGRPCServer()
	if server.Name != "grpc" {
		t.Errorf("Name not set on GRPCServer. got %v want %v", server.Name, "grpc")
	}
}

// Test that graceful stop only waits for drainTimeout, not longer
func TestShutdown(t *testing.T) {
	// Mark test as parallel so it can happen in the background
	t.Parallel()
	// Override drainTimeout for test
	drainTimeout := 500 * time.Millisecond
	viper.Set("drainTimeout", drainTimeout)

	// Create server and client
	lis := bufconn.Listen(1024 * 1024) // use a bufconn listener for testing
	server := NewGRPCServer()
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, address string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatal(err) // could not set up test listener
	}
	defer conn.Close()
	client := NewInfrabinClient(conn)
	// We make a channel for errors as we cannot call t.Fatal inside a non test goroutine
	errs := make(chan error, 1)

	// Start server, call Delay for longer than graceful stop
	go server.ListenAndServe(lis)
	// Wait enough time for server to have started
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	// Call delay with 2 * drainTimeout (1s) in the background
	go func() {
		if _, err := client.Delay(context.Background(), &DelayRequest{Duration: 1}); err != nil {
			errs <- err
		}
	}()
	// Start the shutdown
	server.Shutdown()
	elapsed := time.Since(start).Milliseconds()

	// Check the shutdown waits for at least 500ms, but not the full duration 1s
	if elapsed < drainTimeout.Milliseconds() || elapsed > 1000 {
		t.Fail()
	}
	// There should be an error calling delay as we force stop before
	if err = <-errs; err == nil {
		t.Fail()
	}
}
