package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	infrabinv1 "github.com/maruina/go-infrabin/gen/infrabin/v1"
	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect"
	"github.com/maruina/go-infrabin/internal/aws"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestInfrabinService(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(infrabinv1connect.NewInfrabinServiceHandler(&InfrabinServer{
		STSClient:            aws.FakeSTSClient{},
		ProxyAllowedURLRegex: ".*",
		ProxyHTTPTimeout:     5 * time.Second,
	}))
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
				t.Errorf("got: %v, wanted: %v", result.Msg.Hostname, hostname)
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
				t.Errorf("got: %v, wanted: %v", s, end)
			}
		}
	})

	t.Run("proxy endpoint", func(t *testing.T) {
		headers := map[string]string{
			"Foo": "Bar",
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Foo", headers["Foo"])
			w.WriteHeader(200)
		}))

		for _, client := range clients {
			res, err := client.Proxy(context.Background(), connect.NewRequest(&infrabinv1.ProxyRequest{
				Url:     srv.URL,
				Method:  "GET",
				Headers: headers,
			}))
			if err != nil {
				t.Error("error calling proxy endpoint", err)
			}
			if res.Msg.StatusCode != 200 {
				t.Errorf("error proxy endpoint status code, got: %q, want: %q", res.Msg.StatusCode, 200)
			}
			if res.Msg.Headers["Foo"] != headers["Foo"] {
				t.Errorf("error proxy endpoint headers, got: %v, want: %v", res.Msg.Headers["Foo"], headers["Foo"])
			}
		}
	})

	t.Run("aws assume role endpoint", func(t *testing.T) {
		arn := "arn:aws:sts::123456789012:assumed-role/xaccounts3access/s3-access-example"
		responseString := "AROA3XFRBF535PLBIFPI4:s3-access-example"

		for _, client := range clients {
			_, err := client.AWSAssumeRole(context.Background(), connect.NewRequest(&infrabinv1.AWSAssumeRoleRequest{
				Role: "",
			}))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Error("error calling aws assume role endpoint with empty role", err)
			}

			_, err = client.AWSAssumeRole(context.Background(), connect.NewRequest(&infrabinv1.AWSAssumeRoleRequest{
				Role: "invalid",
			}))
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Errorf("error calling aws assume role endpoint with invalid role got: %v, wanted %v", connect.CodeOf(err), connect.CodeInternal)
			}

			res, err := client.AWSAssumeRole(context.Background(), connect.NewRequest(&infrabinv1.AWSAssumeRoleRequest{
				Role: arn,
			}))
			if err != nil {
				t.Error("error calling aws assume role endpoint", err)
			}
			if !reflect.DeepEqual(res.Msg.AssumedRoleId, responseString) {
				t.Errorf("handler returned unexpected role id: got %v want %s", res.Msg.AssumedRoleId, responseString)
			}
		}
	})

	t.Run("aws get caller identity endpoint", func(t *testing.T) {
		accountId := "123456789012"
		arn := "arn:aws:iam::123456789012:role/my_role"
		userId := "AIDAJQABLZS4A3QDU576Q"

		for _, client := range clients {
			res, err := client.AWSGetCallerIdentity(context.Background(), connect.NewRequest(&infrabinv1.AWSGetCallerIdentityRequest{}))
			if err != nil {
				t.Error("error calling aws get caller identity", err)
			}

			if res.Msg.Account != accountId {
				t.Errorf("handler returned unexpected account: got %v want %s", res.Msg.Account, accountId)
			}
			if res.Msg.Arn != arn {
				t.Errorf("handler returned unexpected arn: got %v want %s", res.Msg.Arn, arn)
			}
			if res.Msg.UserId != userId {
				t.Errorf("handler returned unexpected user_id: got %v want %s", res.Msg.UserId, userId)
			}
		}
	})
}
