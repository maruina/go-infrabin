package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	infrabinv1 "github.com/maruina/go-infrabin/gen/infrabin/v1"
	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
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

	t.Run("env endpoint", func(t *testing.T) {
		envMap := map[string]string{
			"KEY_1": "foo",
			"KEY_2": "bar",
		}

		for k, v := range envMap {
			err := os.Setenv(k, v)
			if err != nil {
				t.Error("error setting environment variable", err)
			}
		}

		for _, client := range clients {
			// If key is empty, return all environment variables
			result, err := client.Env(context.Background(), connect.NewRequest(&infrabinv1.EnvRequest{}))
			if err != nil {
				t.Error("error calling env endpoint", err)
			}
			if len(result.Msg.Environment) <= 2 {
				t.Errorf("not enought environment variables got: %d, want at least 2", len(result.Msg.Environment))
			}

			// If key is set, return that envinronment variable
			// or 404 if it does not exist
			result, err = client.Env(context.Background(), connect.NewRequest(&infrabinv1.EnvRequest{Key: "KEY_1"}))
			if err != nil {
				t.Error("error calling env endpoint", err)
			}
			if result.Msg.Environment["KEY_1"] != envMap["KEY_1"] {
				t.Errorf("got: %v, wanted %v", result.Msg.Environment, envMap["KEY_1"])
			}

			_, err = client.Env(context.Background(), connect.NewRequest(&infrabinv1.EnvRequest{Key: "KEY_NOT_EXIST"}))
			if connect.CodeOf(err) != connect.CodeNotFound {
				t.Errorf("got: %v, wanted: %v", connect.CodeOf(err), connect.CodeNotFound)
			}
		}
	})

	t.Run("delay endpoint", func(t *testing.T) {
		s := time.Second * 1
		for _, client := range clients {
			start := time.Now()
			_, err := client.Delay(context.Background(), connect.NewRequest(&infrabinv1.DelayRequest{
				Duration: durationpb.New(s),
			}))
			if err != nil {
				t.Error("error calling delay endpoint", err)
			}
			end := time.Since(start)
			if end < s {
				t.Errorf("delay error, wanted: %v, got: %v", s, end)
			}
		}
	})
}
