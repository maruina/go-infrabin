package infrabin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/maruina/go-infrabin/internal/helpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	var expected Response
	expected.Hostname = hostname
	expected.Kubernetes = &KubeResponse{
		PodName:   helpers.GetEnv("POD_NAME", ""),
		Namespace: helpers.GetEnv("POD_NAMESPACE", ""),
		PodIp:     helpers.GetEnv("POD_ID", ""),
		NodeName:  helpers.GetEnv("NODE_NAME", ""),
	}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	expectedBytes, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(expectedBytes) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedBytes))
	}
}

func TestFailRootHandler(t *testing.T) {
	if err := os.Setenv("FAIL_ROOT_HANDLER", "true"); err != nil {
		t.Errorf("cannot set environment variable")
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}
	if err = os.Unsetenv("FAIL_ROOT_HANDLER"); err != nil {
		t.Fatal(err)
	}
}

func TestRootHandlerKubernetes(t *testing.T) {
	// Set Kubernetes OS env variables
	podName := "go-infrabin-hjv8k"
	namespace := "kube-system"
	podIP := "172.16.45.234"
	nodeName := "ip-10-51-103-11.eu-west-1.compute.internal"
	os.Setenv("POD_NAME", podName)
	os.Setenv("POD_NAMESPACE", namespace)
	os.Setenv("POD_IP", podIP)
	os.Setenv("NODE_NAME", nodeName)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	var expected Response
	expected.Hostname = hostname
	expected.Kubernetes = &KubeResponse{
		PodName:   podName,
		Namespace: namespace,
		PodIp:     podIP,
		NodeName:  nodeName,
	}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	expectedBytes, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(expectedBytes) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedBytes))
	}
}

func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/liveness", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := Response{Liveness: "pass"}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestDelayHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := Response{Delay: 1}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestDelayHandlerBadRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay/abc", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	expected := status.New(
		codes.InvalidArgument,
		"type mismatch, parameter: duration, error: strconv.ParseInt: parsing \"abc\": invalid syntax",
	)
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	expectedBytes, _ := marshalOptions.Marshal(expected.Proto())

	if !reflect.DeepEqual(rr.Body.Bytes(), expectedBytes) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedBytes))
	}
}

func TestHeadersHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/headers", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-Id", "Test-Header") // Custom header
	req.Header.Set("Accept", "*/*")               // Well known header
	req.Header.Set("Grpc-Metadata-Foo", "bar")    // gRPC metadata

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if s := rr.Code; s != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", s, http.StatusOK)
	}

	expected := Response{Headers: map[string]string{
		"grpcgateway-x-request-id": "Test-Header",
		"grpcgateway-accept":       "*/*",
		"foo":                      "bar",
	}}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestEnvHandler(t *testing.T) {
	if err := os.Setenv("TEST_ENV", "foo"); err != nil {
		t.Errorf("cannot set environment variable")
	}
	req, err := http.NewRequest("GET", "/env/TEST_ENV", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := Response{Env: map[string]string{"TEST_ENV": "foo"}}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestEnvHandlerNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/env/NOT_FOUND", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestProxyHandler(t *testing.T) {
	response, err := json.Marshal(map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("Failed to marshal fake response: %v", err)
	}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(response); err != nil {
			t.Fatalf("Failed to write fake response body: %v", err)
		}
	}))
	defer mockServer.Close()

	body, err := json.Marshal(map[string]interface{}{
		"method":  "POST",
		"url":     mockServer.URL,
		"headers": map[string]string{"Accept": "*/*"},
		"body":    map[string]string{},
	})
	if err != nil {
		t.Fatalf("Failed to make request body: %v", err)
	}

	req, err := http.NewRequest("POST", "/proxy", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", &Config{EnableProxyEndpoint: true}).Server.Handler
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
	if !reflect.DeepEqual(rr.Body.Bytes(), response) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(response))
	}
}

func TestAWSHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("Failed to write fake response body: %v", err)
		}
	}))
	config := &Config{
		EnableProxyEndpoint: true,
		AWSMetadataEndpoint: mockServer.URL,
	}

	req, err := http.NewRequest("GET", "/aws/foo", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", config).Server.Handler
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
	if !reflect.DeepEqual(rr.Body.String(), "{}") {
		t.Errorf("handler returned unexpected body: got %v want {}", rr.Body.String())
	}
}

func TestPromHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewPromServer("test", "", DefaultConfig()).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
