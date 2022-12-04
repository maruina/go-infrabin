package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bufbuild/connect-go"
	infrabinv1 "github.com/maruina/go-infrabin/gen/infrabin/v1"
	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect"
)

func TestInfrabinService(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(infrabinv1connect.NewInfrabinServiceHandler(&InfrabinServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	connectClient := infrabinv1connect.NewInfrabinServiceClient(
		server.Client(),
		server.URL,
	)
	grcpClient := infrabinv1connect.NewInfrabinServiceClient(
		server.Client(),
		server.URL,
		connect.WithGRPC(),
	)
	clients := []infrabinv1connect.InfrabinServiceClient{connectClient, grcpClient}

	t.Run("root endpoint", func(t *testing.T) {
		hostname, err := os.Hostname()
		if err != nil {
			t.Error("error getting hostname", err)
		}
		for _, client := range clients {
			result, err := client.Root(context.Background(), connect.NewRequest(&infrabinv1.RootRequest{}))
			if err != nil {
				t.Error("error calling root endpoint", err)
			}
			if result.Msg.Hostname != hostname {
				t.Errorf("hostname error, got: %v, want: %v", result.Msg.Hostname, hostname)
			}
		}
	})
}
