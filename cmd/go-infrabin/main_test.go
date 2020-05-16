package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	helpers "github.com/maruina/go-infrabin/internal/helpers"
)

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RootHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	var expected helpers.Response
	expected.Hostname = hostname
	expected.KubeResponse = &helpers.KubeResponse{
		PodName:   helpers.GetEnv("POD_NAME", ""),
		Namespace: helpers.GetEnv("POD_NAMESPACE", ""),
		PodIP:     helpers.GetEnv("POD_ID", ""),
		NodeName:  helpers.GetEnv("NODE_NAME", ""),
	}
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
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
	handler := http.HandlerFunc(RootHandler)
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

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RootHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	var expected helpers.Response
	expected.Hostname = hostname
	expected.KubeResponse = &helpers.KubeResponse{
		PodName:   podName,
		Namespace: namespace,
		PodIP:     podIP,
		NodeName:  nodeName,
	}
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}

func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthcheck/liveness", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LivenessHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expected helpers.Response
	expected.Liveness = "pass"
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}

func TestDelayHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay", nil)
	req = mux.SetURLVars(req, map[string]string{"seconds": "1"})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DelayHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expected helpers.Response
	expected.Delay = "1"
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}

func TestDelayHandlerBadRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/delay", nil)
	req = mux.SetURLVars(req, map[string]string{"seconds": "abc"})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DelayHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	var expected helpers.Response
	expected.Error = "cannot convert seconds to integer"
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}

func TestHeadersHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/headers", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-Id", "Test-Header")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HeadersHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expected helpers.Response
	expected.Headers = &http.Header{
		"X-Request-Id": []string{"Test-Header"},
	}
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
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

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(EnvHandler)
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

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(EnvHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}
