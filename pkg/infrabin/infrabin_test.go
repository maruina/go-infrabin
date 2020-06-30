package infrabin

import (
	"github.com/gorilla/mux"
	"github.com/maruina/go-infrabin/internal/helpers"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rootHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Root").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(rootHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
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
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(data))
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

	rootHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Root").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(rootHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusServiceUnavailable)
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

	rootHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Root").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(rootHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
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
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(data))
	}
}


func TestDelayHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay", nil)
	req = mux.SetURLVars(req, map[string]string{"seconds": "1"})
	if err != nil {
		t.Fatal(err)
	}

	delayHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Delay").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(delayHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := Response{Delay: 1}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data){
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(data))
	}
}



func TestDelayHandlerBadRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay", nil)
	req = mux.SetURLVars(req, map[string]string{"seconds": "abc"})
	if err != nil {
		t.Fatal(err)
	}

	delayHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Delay").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(delayHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	expected := Response{Error: "Failed to build request: strconv.Atoi: parsing \"abc\": invalid syntax"}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(data))
	}
}



func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthcheck/liveness", nil)
	if err != nil {
		t.Fatal(err)
	}

	livenessHandler := NewAdminServer().Server.Handler.(*mux.Router).Get("Liveness").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(livenessHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := Response{Liveness: "pass"}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(data))
	}
}



func TestEnvHandler(t *testing.T) {
	if err := os.Setenv("TEST_ENV", "foo"); err != nil {
		t.Errorf("cannot set environment variable")
	}
	req, err := http.NewRequest("GET", "/env", nil)
	req = mux.SetURLVars(req, map[string]string{"env_var": "TEST_ENV"})
	if err != nil {
		t.Fatal(err)
	}

	envHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Env").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(envHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expected helpers.Response
	expected.Env = map[string]string{
		"TEST_ENV": "foo",
	}
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}

func TestEnvHandlerNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/env", nil)
	req = mux.SetURLVars(req, map[string]string{"env_var": "NOT_FOUND"})
	if err != nil {
		t.Fatal(err)
	}

	envHandler := NewHTTPServer().Server.Handler.(*mux.Router).Get("Env").GetHandler().ServeHTTP
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(envHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusServiceUnavailable)
	}
}
