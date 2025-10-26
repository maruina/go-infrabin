package infrabin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	"github.com/google/go-cmp/cmp"
	"github.com/maruina/go-infrabin/internal/aws"
	"github.com/maruina/go-infrabin/internal/helpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"
)

func newHTTPInfrabinHandler() http.Handler {
	srv, err := NewHTTPServer(
		"test",
		RegisterInfrabin("/", &InfrabinService{
			STSClient: aws.FakeSTSClient{},
			// intermittentErrorsCounter uses atomic.Int32 zero value
		}),
	)
	if err != nil {
		panic(err) // Panic in test helper is acceptable
	}
	return srv.Server.Handler
}

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
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
		Region:    helpers.GetEnv("REGION", ""),
	}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	expectedBytes, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(expectedBytes) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedBytes))
	}
}

func TestFailRootHandler(t *testing.T) {
	t.Setenv("FAIL_ROOT_HANDLER", "true")
	req := httptest.NewRequest("GET", "/", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}
}

func TestRootHandlerKubernetes(t *testing.T) {
	// Set Kubernetes OS env variables
	podName := "go-infrabin-hjv8k"
	namespace := "kube-system"
	podIP := "172.16.45.234"
	nodeName := "ip-10-51-103-11.eu-west-1.compute.internal"
	region := "eu-west-1"
	t.Setenv("POD_NAME", podName)
	t.Setenv("POD_NAMESPACE", namespace)
	t.Setenv("POD_IP", podIP)
	t.Setenv("NODE_NAME", nodeName)
	t.Setenv("REGION", region)

	req := httptest.NewRequest("GET", "/", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
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
		Region:    region,
	}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	expectedBytes, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(expectedBytes) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedBytes))
	}
}

func TestDelayHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/delay/1", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
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
	req := httptest.NewRequest("GET", "/delay/abc", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
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
	req := httptest.NewRequest("GET", "/headers", nil)
	req.Header.Set("X-Request-Id", "Test-Header") // Custom header
	req.Header.Set("Accept", "*/*")               // Well known header
	req.Header.Set("Grpc-Metadata-Foo", "bar")    // gRPC metadata

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if s := rr.Code; s != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", s, http.StatusOK)
	}

	expected := Response{Headers: map[string]string{
		"grpcgateway-x-request-id": "Test-Header",
		"grpcgateway-accept":       "*/*",
		"foo":                      "bar",
		"x-forwarded-for":          "192.0.2.1",   // from httptest.NewRequest
		"x-forwarded-host":         "example.com", // from httptest.NewRequest
	}}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestEnvHandler(t *testing.T) {
	t.Setenv("TEST_ENV", "foo")
	req := httptest.NewRequest("GET", "/env/TEST_ENV", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
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
	req := httptest.NewRequest("GET", "/env/NOT_FOUND", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestProxyHandlerRegexpAllowURL(t *testing.T) {
	// Set the Proxy to true for testing
	viper.Set("proxyEndpoint", true)

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

	// Allow the mock server URL
	viper.Set("proxyAllowRegexp", mockServer.URL)

	req := httptest.NewRequest("POST", "/proxy", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
	if !reflect.DeepEqual(rr.Body.Bytes(), response) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(response))
	}
}

func TestProxyHandlerRegexpDenyURL(t *testing.T) {
	// Set the Proxy to true for testing
	viper.Set("proxyEndpoint", true)
	viper.Set("proxyAllowRegexp", "fakeurl")

	body, err := json.Marshal(map[string]interface{}{
		"method":  "POST",
		"url":     "http://www.example.org",
		"headers": map[string]string{"Accept": "*/*"},
		"body":    map[string]string{},
	})
	if err != nil {
		t.Fatalf("Failed to make request body: %v", err)
	}

	req := httptest.NewRequest("POST", "/proxy", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusBadRequest)
	}
}

func TestAWSMetadataHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("Failed to write fake response body: %v", err)
		}
	}))

	viper.Set("proxyEndpoint", true)
	viper.Set("proxyAllowRegexp", ".*")
	viper.Set("awsMetadataEndpoint", mockServer.URL)

	req := httptest.NewRequest("GET", "/aws/metadata/foo", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
	if !reflect.DeepEqual(rr.Body.String(), "{}") {
		t.Errorf("handler returned unexpected body: got %v want {}", rr.Body.String())
	}
}

func TestPromHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics", nil)

	rr := httptest.NewRecorder()
	srv, err := NewHTTPServer(
		"test-prom",
		RegisterHandler("/", promhttp.Handler()),
	)
	if err != nil {
		t.Fatalf("Failed to create HTTP server: %v", err)
	}
	handler := srv.Server.Handler
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestAnyHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/any/foo/bar", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	expected := Response{Path: "foo/bar"}
	marshalOptions := protojson.MarshalOptions{UseProtoNames: true}
	data, _ := marshalOptions.Marshal(&expected)

	if rr.Body.String() != string(data) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(data))
	}
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestAWSAssumeHandler(t *testing.T) {
	arn := "arn:aws:sts::123456789012:assumed-role/xaccounts3access/s3-access-example"
	req := httptest.NewRequest("GET", fmt.Sprintf("/aws/assume/%s", arn), nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
	responseString := "{\"assumedRoleId\":\"AROA3XFRBF535PLBIFPI4:s3-access-example\"}"
	if !reflect.DeepEqual(rr.Body.String(), responseString) {
		t.Errorf("handler returned unexpected body: got %v want %s", rr.Body.String(), responseString)
	}
}

func TestAWSAssumeHandlerWithEmptyRole(t *testing.T) {
	req := httptest.NewRequest("GET", "/aws/assume/", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusBadRequest)
	}
}

func TestAWSAssumeHandlerWithInvalidRole(t *testing.T) {
	req := httptest.NewRequest("GET", "/aws/assume/bad_role", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusInternalServerError)
	}
}

func TestAWSGetCallerIdentityHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/aws/get-caller-identity", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// See https://github.com/golang/protobuf/issues/1121
	responseString := "{\"getCallerIdentity\":{\"account\":\"123456789012\",\"arn\":\"arn:aws:iam::123456789012:role/my_role\",\"user_id\":\"AIDAJQABLZS4A3QDU576Q\"}}"
	var responseJSON map[string]interface{}
	if err := json.Unmarshal([]byte(responseString), &responseJSON); err != nil {
		t.Fatalf("failed to parse responseString %v: %v", responseString, err)
	}

	var rrJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &rrJSON); err != nil {
		t.Fatalf("failed to parse responseRecorder body %v: %v", rr.Body.String(), err)
	}

	if diff := cmp.Diff(responseJSON, rrJSON); diff != "" {
		t.Errorf("unexpected difference (-want +got):\n%s", diff)
	}
}

func TestIntermittentHandler(t *testing.T) {
	viper.Set("intermittentErrors", 2)
	req := httptest.NewRequest("GET", "/intermittent", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()

	// First request should be 503
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusServiceUnavailable)
	}

	responseString := "{\"code\":14,\"message\":\"2 errors left\"}"
	var responseJSON map[string]interface{}
	if err := json.Unmarshal([]byte(responseString), &responseJSON); err != nil {
		t.Fatalf("failed to parse responseString %v: %v", responseString, err)
	}

	var rrJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &rrJSON); err != nil {
		t.Fatalf("failed to parse responseRecorder body %v: %v", rr.Body.String(), err)
	}

	if diff := cmp.Diff(responseJSON, rrJSON); diff != "" {
		t.Errorf("unexpected difference (-want +got):\n%s", diff)
	}

	// Second request should be 503
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusServiceUnavailable)
	}

	responseString = "{\"code\":14,\"message\":\"1 errors left\"}"
	if err := json.Unmarshal([]byte(responseString), &responseJSON); err != nil {
		t.Fatalf("failed to parse responseString %v: %v", responseString, err)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &rrJSON); err != nil {
		t.Fatalf("failed to parse responseRecorder body %v: %v", rr.Body.String(), err)
	}

	if diff := cmp.Diff(responseJSON, rrJSON); diff != "" {
		t.Errorf("unexpected difference (-want +got):\n%s", diff)
	}

	// Third request should be 200
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	responseString = "{\"intermittent\":{\"intermittent_errors\":2}}"
	if err := json.Unmarshal([]byte(responseString), &responseJSON); err != nil {
		t.Fatalf("failed to parse responseString %v: %v", responseString, err)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &rrJSON); err != nil {
		t.Fatalf("failed to parse responseRecorder body %v: %v", rr.Body.String(), err)
	}

	if diff := cmp.Diff(responseJSON, rrJSON); diff != "" {
		t.Errorf("unexpected difference (-want +got):\n%s", diff)
	}

}

func TestEgressDNSHandler(t *testing.T) {
	testCases := []struct {
		name           string
		host           string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "valid hostname",
			host:           "google.com",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "localhost",
			host:           "localhost",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid hostname",
			host:           "this-domain-does-not-exist-12345.invalid",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/egress/dns/%s", tc.host), nil)

			rr := httptest.NewRecorder()
			handler := newHTTPInfrabinHandler()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tc.expectedStatus)
			}

			// Parse as generic JSON to handle protobuf's int64-as-string
			var responseJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Protobuf omits false boolean values from JSON, so absence means false
			success, ok := responseJSON["success"].(bool)
			if !ok {
				success = false // Default value when omitted
			}
			if success != tc.expectSuccess {
				errorMsg := ""
				if e, ok := responseJSON["error"].(string); ok {
					errorMsg = e
				}
				t.Errorf("expected success=%v, got success=%v, error=%s", tc.expectSuccess, success, errorMsg)
			}

			target, ok := responseJSON["target"].(string)
			if !ok {
				t.Fatal("target field not found or not a string")
			}
			if target != tc.host {
				t.Errorf("expected target=%s, got target=%s", tc.host, target)
			}

			if tc.expectSuccess {
				resolvedIps, ok := responseJSON["resolvedIps"].([]interface{})
				if !ok || len(resolvedIps) == 0 {
					t.Error("expected at least one resolved IP for successful DNS lookup")
				}
			}

			// durationMs may be omitted by protobuf JSON marshaling when it's 0 (very fast localhost DNS)
			// Just verify it's a valid number if present
			if durationMs, ok := responseJSON["durationMs"]; ok {
				if _, isFloat := durationMs.(float64); !isFloat {
					if _, isString := durationMs.(string); !isString {
						t.Errorf("expected durationMs to be a number, got %T: %v", durationMs, durationMs)
					}
				}
			}
		})
	}
}

func TestEgressHTTPHandler(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Extract host:port from mock server URL (remove http://)
	target := mockServer.URL[7:] // Remove "http://"

	req := httptest.NewRequest("GET", fmt.Sprintf("/egress/http/%s", target), nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v, body: %s", err, rr.Body.String())
	}

	success, ok := responseJSON["success"].(bool)
	if !ok {
		success = false // Protobuf omits false
	}
	if !success {
		errorMsg := ""
		if e, ok := responseJSON["error"].(string); ok {
			errorMsg = e
		}
		t.Errorf("expected success=true, got success=%v, error=%s", success, errorMsg)
	}

	// statusCode can be float64 or string depending on JSON unmarshaling
	if statusCode, ok := responseJSON["statusCode"].(float64); ok {
		if int(statusCode) != 200 {
			t.Errorf("expected status code 200, got %v", statusCode)
		}
	}

	// durationMs may be omitted by protobuf JSON marshaling when it's 0 (very fast localhost connections)
	// Just verify it's a valid number if present
	if durationMs, ok := responseJSON["durationMs"]; ok {
		if _, isFloat := durationMs.(float64); !isFloat {
			if _, isString := durationMs.(string); !isString {
				t.Errorf("expected durationMs to be a number, got %T: %v", durationMs, durationMs)
			}
		}
	}
}

func TestEgressHTTPHandlerFailure(t *testing.T) {
	// Use an invalid port to force connection failure
	req := httptest.NewRequest("GET", "/egress/http/localhost:1", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	success, _ := responseJSON["success"].(bool)
	if success {
		t.Error("expected success=false for connection to invalid port")
	}

	if _, ok := responseJSON["error"].(string); !ok {
		t.Error("expected error message to be set")
	}
}

func TestEgressHTTPSHandler(t *testing.T) {
	// Create a mock HTTPS server
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Extract host:port from mock server URL (remove https://)
	target := mockServer.URL[8:] // Remove "https://"

	// Note: This will fail certificate verification since it's a test server
	// That's expected behavior - we test insecure mode separately
	req := httptest.NewRequest("GET", fmt.Sprintf("/egress/https/%s", target), nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// httptest.NewTLSServer creates a self-signed cert, so this should fail
	success, _ := responseJSON["success"].(bool)
	if success {
		t.Error("expected success=false for self-signed certificate without insecure mode")
	}

	if _, ok := responseJSON["error"].(string); !ok {
		t.Error("expected error message about certificate verification")
	}
}

func TestEgressHTTPSInsecureHandler(t *testing.T) {
	// Create a mock HTTPS server
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Extract host:port from mock server URL (remove https://)
	target := mockServer.URL[8:] // Remove "https://"

	req := httptest.NewRequest("GET", fmt.Sprintf("/egress/https/insecure/%s", target), nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// With insecure mode, this should succeed
	success, _ := responseJSON["success"].(bool)
	if !success {
		errorMsg := ""
		if e, ok := responseJSON["error"].(string); ok {
			errorMsg = e
		}
		t.Errorf("expected success=true with insecure mode, got success=%v, error=%s", success, errorMsg)
	}

	if statusCode, ok := responseJSON["statusCode"].(float64); ok {
		if int(statusCode) != 200 {
			t.Errorf("expected status code 200, got %v", statusCode)
		}
	}

	if _, ok := responseJSON["durationMs"]; !ok {
		t.Error("expected durationMs field to be present")
	}
}

func TestEgressHTTPDefaultPort(t *testing.T) {
	// Test that default port 80 is added when not specified
	// This will fail to connect, but we can verify the target includes the port
	req := httptest.NewRequest("GET", "/egress/http/example.com", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	target, _ := responseJSON["target"].(string)
	if target != "example.com:80" {
		t.Errorf("expected target to include default port 80, got %s", target)
	}
}

func TestEgressHTTPSDefaultPort(t *testing.T) {
	// Test that default port 443 is added when not specified
	req := httptest.NewRequest("GET", "/egress/https/example.com", nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	target, _ := responseJSON["target"].(string)
	if target != "example.com:443" {
		t.Errorf("expected target to include default port 443, got %s", target)
	}
}

func TestEgressHTTPWithCustomDNS(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Extract host:port from mock server URL (remove http://)
	target := mockServer.URL[7:] // Remove "http://"

	// Test with custom DNS server specified (though it won't actually be used for localhost)
	// This test verifies the @ syntax is correctly parsed
	targetWithDNS := target + "@8.8.8.8:53"
	req := httptest.NewRequest("GET", fmt.Sprintf("/egress/http/%s", targetWithDNS), nil)

	rr := httptest.NewRecorder()
	handler := newHTTPInfrabinHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var responseJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &responseJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify the target in response is the hostPort (without DNS server)
	// The DNS server part is used for resolution but not included in the response target
	responseTarget, _ := responseJSON["target"].(string)
	if responseTarget != target {
		t.Errorf("expected target %s in response (without DNS server), got %s", target, responseTarget)
	}

	// Verify success
	success, _ := responseJSON["success"].(bool)
	if !success {
		errorMsg := ""
		if e, ok := responseJSON["error"].(string); ok {
			errorMsg = e
		}
		t.Errorf("expected success=true, got success=%v, error=%s", success, errorMsg)
	}
}

func TestParseTargetAndDNS(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		wantHost string
		wantDNS  string
	}{
		{
			name:     "host and port with DNS",
			target:   "example.com:443@8.8.8.8:53",
			wantHost: "example.com:443",
			wantDNS:  "8.8.8.8:53",
		},
		{
			name:     "host only with DNS",
			target:   "example.com@8.8.8.8:53",
			wantHost: "example.com",
			wantDNS:  "8.8.8.8:53",
		},
		{
			name:     "host and port without DNS",
			target:   "example.com:443",
			wantHost: "example.com:443",
			wantDNS:  "",
		},
		{
			name:     "host only without DNS",
			target:   "example.com",
			wantHost: "example.com",
			wantDNS:  "",
		},
		{
			name:     "IP address with port and DNS",
			target:   "192.168.1.1:8080@1.1.1.1:53",
			wantHost: "192.168.1.1:8080",
			wantDNS:  "1.1.1.1:53",
		},
		{
			name:     "multiple @ symbols takes first as separator",
			target:   "host:443@dns1:53@dns2:53",
			wantHost: "host:443",
			wantDNS:  "dns1:53", // Only first @ is separator, rest are part of subsequent string
		},
		{
			name:     "empty string",
			target:   "",
			wantHost: "",
			wantDNS:  "",
		},
		{
			name:     "only @ symbol",
			target:   "@",
			wantHost: "",
			wantDNS:  "",
		},
		{
			name:     "host with trailing @",
			target:   "example.com@",
			wantHost: "example.com",
			wantDNS:  "",
		},
		{
			name:     "DNS without host",
			target:   "@8.8.8.8:53",
			wantHost: "",
			wantDNS:  "8.8.8.8:53",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotHost, gotDNS := parseTargetAndDNS(tc.target)
			if gotHost != tc.wantHost {
				t.Errorf("parseTargetAndDNS(%q) host = %q, want %q", tc.target, gotHost, tc.wantHost)
			}
			if gotDNS != tc.wantDNS {
				t.Errorf("parseTargetAndDNS(%q) DNS = %q, want %q", tc.target, gotDNS, tc.wantDNS)
			}
		})
	}
}

func TestHTTPMetricsCollection(t *testing.T) {
	// Create HTTP server with metrics middleware
	handler := newHTTPInfrabinHandler()

	// Set up test environment variable
	t.Setenv("TEST_ENV", "test_value")

	// Test cases that exercise different endpoints and status codes
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedRoute  string
	}{
		{
			name:           "headers endpoint",
			path:           "/headers",
			expectedStatus: http.StatusOK,
			expectedRoute:  "/headers",
		},
		{
			name:           "delay endpoint with parameter",
			path:           "/delay/2",
			expectedStatus: http.StatusOK,
			expectedRoute:  "/delay/{duration}",
		},
		{
			name:           "env endpoint",
			path:           "/env/TEST_ENV",
			expectedStatus: http.StatusOK,
			expectedRoute:  "/env/{env_var}",
		},
		{
			name:           "root endpoint",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedRoute:  "/",
		},
	}

	// Execute test requests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("%s: got status %d, want %d", tc.name, rr.Code, tc.expectedStatus)
			}
		})
	}

	// Verify metrics are collected by checking the prometheus registry
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	// Use the prometheus handler to get metrics
	promHandler := promhttp.Handler()
	promHandler.ServeHTTP(rr, req)

	body := rr.Body.String()

	// Check for HTTP metrics presence
	expectedMetrics := []string{
		"infrabin_http_request_duration_seconds",
		"infrabin_http_requests_total",
	}

	for _, metric := range expectedMetrics {
		if !bytes.Contains([]byte(body), []byte(metric)) {
			t.Errorf("Expected metric %s not found in output", metric)
		}
	}

	// Verify that we have metrics with handler names (matching gRPC metric style)
	expectedLabels := []string{
		`handler="headers"`,
		`handler="delay"`,
		`handler="env"`,
		`handler="root"`,
	}

	for _, label := range expectedLabels {
		if !bytes.Contains([]byte(body), []byte(label)) {
			t.Errorf("Expected label %s not found in metrics output", label)
		}
	}
}
