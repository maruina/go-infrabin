package infrabin

import (
	"encoding/json"
	"github.com/maruina/go-infrabin/internal/helpers"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	// Cast to string and replace spaces. No need to replace if grpc-gateway uses protojson
	expectedStr := strings.Replace(string(expectedBytes), " ", "", -1)

	if rr.Body.String() != expectedStr {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedStr)
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	// Cast to string and replace spaces. No need to replace if grpc-gateway uses protojson
	expectedStr := strings.Replace(string(expectedBytes), " ", "", -1)

	if rr.Body.String() != expectedStr {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedStr)
	}
}

func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/liveness", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	expected := map[string]interface{}{
		"code": 3,
		"error": "type mismatch, parameter: duration, error: strconv.ParseInt: parsing \"abc\": invalid syntax",
		"message": "type mismatch, parameter: duration, error: strconv.ParseInt: parsing \"abc\": invalid syntax",
	}
	var got map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &got)
	if err != nil {
		t.Errorf("Cannot unmarshal response: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", got, expected)
	}
}

func TestHeadersHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/headers", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-Id", "Test-Header")

	rr := httptest.NewRecorder()
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := Response{Headers: map[string]string{"X-Request-Id": "Test-Header"}}
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
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
	handler := NewHTTPServer("test", "", &Config{}).Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}
