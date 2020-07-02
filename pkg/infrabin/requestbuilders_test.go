package infrabin

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"net/http"
	"testing"
)

func makeRequestBody(t testing.TB, j map[string]interface{}) *bytes.Buffer{
	b, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func TestBuildEmpty(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Failed to build http.Request: %v", err)
	}

	expected := &Empty{}

	if msg, err := BuildEmpty(request); err != nil {
		t.Errorf("BuildEmpty returned an error, %v", err)
	} else if !proto.Equal(msg, expected) {
		t.Errorf("Expected %v got %v", expected, msg)
	}
}


func TestBuildEnvRequest(t *testing.T) {
	request, err := http.NewRequest("GET", "/env", nil)
	if err != nil {
		t.Fatalf("Failed to build http.Request: %v", err)
	}
	request = mux.SetURLVars(request, map[string]string{"env_var": "TEST_ENV"})

	expected := &EnvRequest{EnvVar: "TEST_ENV"}

	if msg, err := BuildEnvRequest(request); err != nil {
		t.Errorf("BuildEnvRequest returned an error, %v", err)
	} else if !proto.Equal(msg, expected) {
		t.Errorf("Expected %v got %v", expected, msg)
	}
}


func TestBuildHeadersRequest(t *testing.T) {
	request, err := http.NewRequest("GET", "/headers", nil)
	if err != nil {
		t.Fatalf("Failed to build http.Request: %v", err)
	}
	request.Header.Set("X-Request-Id", "Test-Header")

	expected := &HeadersRequest{Headers: map[string]string{"X-Request-Id": "Test-Header"}}

	if msg, err := BuildHeadersRequest(request); err != nil {
		t.Errorf("BuildHeadersRequest returned an error, %v", err)
	} else if !proto.Equal(msg, expected) {
		t.Errorf("Expected %v got %v", expected, msg)
	}
}


func TestBuildProxyRequest(t *testing.T) {
	body := makeRequestBody(t, map[string]interface{}{
		"method": "POST",
		"url": "http://httpbin.org/post",
		"body": map[string]string{"a": "b"},
		"headers": map[string]string{"X-Request-Id": "Test-Header"},
	})
	request, err := http.NewRequest("GET", "/proxy", body)
	if err != nil {
		t.Fatalf("Failed to build http.Request: %v", err)
	}

	structBody, err := structpb.NewStruct(map[string]interface{}{"a": "b"})
	if err != nil {
		t.Fatalf("Failed to create Struct: %v", err)
	}
	expected := &ProxyRequest{
		Method: "POST",
		Url: "http://httpbin.org/post",
		Body: structBody,
		Headers: map[string]string{"X-Request-Id": "Test-Header"},
	}

	if msg, err := BuildProxyRequest(request); err != nil {
		t.Errorf("BuildProxyRequest returned an error, %v", err)
	} else if !proto.Equal(msg, expected) {
		t.Errorf("Expected %v got %v", expected, msg)
	}
}

func TestBuildAWSRequest(t *testing.T) {
	request, err := http.NewRequest("GET", "/aws", nil)
	if err != nil {
		t.Fatalf("Failed to build http.Request: %v", err)
	}
	request = mux.SetURLVars(request, map[string]string{"path": "foo/bar"})

	expected := &AWSRequest{Path: "foo/bar"}

	if msg, err := BuildAWSRequest(request); err != nil {
		t.Errorf("BuildAWSRequest returned an error, %v", err)
	} else if !proto.Equal(msg, expected) {
		t.Errorf("Expected %v got %v", expected, msg)
	}
}
