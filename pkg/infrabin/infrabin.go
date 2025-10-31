// Package infrabin provides HTTP and gRPC server implementations for simulating
// infrastructure endpoints in testing environments. It supports endpoints for
// testing delays, headers, environment variables, AWS metadata, and more.
package infrabin

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/maruina/go-infrabin/internal/aws"
	"github.com/maruina/go-infrabin/internal/helpers"
	"github.com/mazen160/go-random"
	"github.com/spf13/viper"
)

// InfrabinService implements the Infrabin gRPC service.
// It must embed UnimplementedInfrabinServer for protogen-gen-go-grpc compatibility.
type InfrabinService struct {
	UnimplementedInfrabinServer
	STSClient                 aws.STSClient
	HealthService             HealthService
	intermittentErrorsCounter atomic.Int32
	K8sClient                 K8sClient // Optional: nil if crossaz endpoint disabled
}

// K8sClient defines the interface for Kubernetes operations needed by CrossAZ.
// This interface allows for easier testing and mocking.
type K8sClient interface {
	DiscoverPods(ctx context.Context, labelSelector string) ([]K8sPodInfo, error)
}

// K8sPodInfo contains essential information about a discovered pod.
// This is a local type to avoid direct dependency on internal/k8s in the interface.
type K8sPodInfo struct {
	Name             string
	IP               string
	AvailabilityZone string
}

// HealthService defines the interface for managing health check status.
// This interface allows setting the serving status for different service names.
type HealthService interface {
	SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus)
}

// derefString safely dereferences a string pointer, returning empty string if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// validateDNSServerAddress validates and normalizes a DNS server address.
// If the address doesn't include a port, it adds the default port 53.
// Returns the normalized address or an error if the address format is invalid.
func validateDNSServerAddress(dnsServer string) (string, error) {
	if dnsServer == "" {
		return "", nil
	}

	// Check if port exists using SplitHostPort (handles IPv6 correctly)
	_, _, err := net.SplitHostPort(dnsServer)
	if err != nil {
		// No port specified, add default using JoinHostPort (handles IPv6)
		dnsServer = net.JoinHostPort(dnsServer, "53")
	}
	return dnsServer, nil
}

func (s *InfrabinService) Root(ctx context.Context, _ *Empty) (*Response, error) {
	fail := helpers.GetEnv("FAIL_ROOT_HANDLER", "")
	if fail != "" {
		return nil, status.Errorf(codes.Unavailable, "root handler is configured to fail via FAIL_ROOT_HANDLER=%s", fail)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get hostname: %v", err)
	}

	var resp Response
	resp.Hostname = hostname
	// Take kubernetes info from a couple of _common_ environment variables
	resp.Kubernetes = &KubeResponse{
		PodName:     helpers.GetEnv("POD_NAME", "K8S_POD_NAME", ""),
		Namespace:   helpers.GetEnv("POD_NAMESPACE", "K8S_NAMESPACE", ""),
		PodIp:       helpers.GetEnv("POD_IP", "K8S_POD_IP", ""),
		NodeName:    helpers.GetEnv("NODE_NAME", "K8S_NODE_NAME", ""),
		ClusterName: helpers.GetEnv("CLUSTER_NAME", "K8S_CLUSTER_NAME", ""),
		Region:      helpers.GetEnv("REGION", "AWS_REGION", "FUNCTION_REGION", ""),
	}
	return &resp, nil
}

func (s *InfrabinService) Delay(ctx context.Context, request *DelayRequest) (*Response, error) {
	maxDelay := viper.GetDuration("maxDelay")
	requestDuration := time.Duration(request.Duration) * time.Second

	duration := helpers.MinDuration(requestDuration, maxDelay)

	// Respect context cancellation during the delay
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return &Response{Delay: int32(duration.Seconds())}, nil
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}
}

func (s *InfrabinService) Env(ctx context.Context, request *EnvRequest) (*Response, error) {
	value := helpers.GetEnv(request.EnvVar, "")
	if value == "" {
		return nil, status.Errorf(codes.NotFound, "no env var named %s", request.EnvVar)
	}
	return &Response{Env: map[string]string{request.EnvVar: value}}, nil
}

func (s *InfrabinService) Headers(ctx context.Context, request *HeadersRequest) (*Response, error) {
	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	for key := range md {
		request.Headers[key] = strings.Join(md.Get(key), ",")
	}
	return &Response{Headers: request.Headers}, nil
}

func (s *InfrabinService) Proxy(ctx context.Context, request *ProxyRequest) (*structpb.Struct, error) {
	if !viper.GetBool("proxyEndpoint") {
		return nil, status.Errorf(codes.Unimplemented, "Proxy endpoint disabled. Enabled with --enable-proxy-endpoint")
	}

	// Compile the regexp
	exp := viper.GetString("proxyAllowRegexp")
	r, err := regexp.Compile(exp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to compile %s regexp: %v", exp, err)
	}

	// Convert Struct into json []byte
	requestBody, err := request.Body.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to marshal downstream request body: %v", err)
	}

	// Check if the target URL is allowed
	if !r.MatchString(request.Url) {
		return nil, status.Errorf(codes.InvalidArgument, "Unable to build request as the target URL %s is blocked by the regexp %s", request.Url, exp)
	}

	// Make upstream request from incoming request
	req, err := http.NewRequestWithContext(ctx, request.Method, request.Url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to build request: %v", err)
	}
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// Send http request with configured timeout
	timeout := viper.GetDuration("egressTimeout")
	client := http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach %s: %v", request.Url, err)
	}

	// Read request body and close it
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read upstream response body: %v", err)
	}
	if err = resp.Body.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close upstream response: %v", err)
	}

	// Convert []bytes into json struct
	var response structpb.Struct
	if err := response.UnmarshalJSON(body); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create struct from upstream response json: %v", err)
	}
	return &response, nil
}

func (s *InfrabinService) AWSMetadata(ctx context.Context, request *AWSMetadataRequest) (*structpb.Struct, error) {
	if request.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "path must not be empty")
	}

	u, err := url.Parse(viper.GetString("awsMetadataEndpoint"))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "s.Config.AWSMetadataEndpoint invalid: %v", err)
	}

	u.Path = request.Path
	return s.Proxy(ctx, &ProxyRequest{Method: "GET", Url: u.String()})
}

func (s *InfrabinService) Any(ctx context.Context, request *AnyRequest) (*Response, error) {
	return &Response{Path: request.Path}, nil
}

func (s *InfrabinService) AWSAssume(ctx context.Context, request *AWSAssumeRequest) (*Response, error) {
	if request.Role == "" {
		return nil, status.Errorf(codes.InvalidArgument, "role must not be empty")
	}

	roleId, err := aws.STSAssumeRole(ctx, s.STSClient, request.Role, "aws-assume-session-go-infrabin")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assume AWS IAM role: %v", err)
	}

	return &Response{AssumedRoleId: roleId}, nil
}

func (s *InfrabinService) AWSGetCallerIdentity(ctx context.Context, _ *Empty) (*Response, error) {
	response, err := aws.STSGetCallerIdentity(ctx, s.STSClient)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get AWS caller identity: %v", err)
	}

	// Safely dereference pointers with default values to prevent nil pointer panics
	return &Response{
		GetCallerIdentity: &GetCallerIdentityResponse{
			Account: derefString(response.Account),
			Arn:     derefString(response.Arn),
			UserId:  derefString(response.UserId),
		},
	}, nil
}

func (s *InfrabinService) Intermittent(ctx context.Context, _ *Empty) (*Response, error) {
	maxErrs := viper.GetInt32("intermittentErrors")

	counter := s.intermittentErrorsCounter.Add(1)
	if counter <= maxErrs {
		return nil, status.Errorf(codes.Unavailable, "%d errors left", maxErrs-counter+1)
	}

	// Use CompareAndSwap to atomically reset counter only if it's still at the value we read.
	// This prevents race conditions where multiple goroutines might try to reset simultaneously.
	// We intentionally ignore the boolean return value - if another goroutine already reset it, that's fine.
	s.intermittentErrorsCounter.CompareAndSwap(counter, 0)
	return &Response{
		Intermittent: &IntermittentResponse{
			IntermittentErrors: maxErrs,
		},
	}, nil
}

func (s *InfrabinService) RandomData(ctx context.Context, request *RandomDataRequest) (*Response, error) {
	response, err := random.Bytes(int(request.GetPath()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate random data: %v", err)
	}
	return &Response{
		RandomData: &RandomDataResponse{
			Data: response,
		},
	}, nil
}

func (s *InfrabinService) EgressDNS(ctx context.Context, request *EgressDNSRequest) (*EgressResponse, error) {
	if request.Host == "" {
		return nil, status.Errorf(codes.InvalidArgument, "host must not be empty")
	}

	hostname, dnsServer := parseTargetAndDNS(request.Host)
	resolver, err := s.createDNSResolver(dnsServer)
	if err != nil {
		return &EgressResponse{
			Success: false,
			Error:   err.Error(),
			Target:  hostname,
		}, nil
	}

	start := time.Now()
	ips, err := resolver.LookupHost(ctx, hostname)
	duration := time.Since(start)

	if err != nil {
		return &EgressResponse{
			Success:    false,
			Error:      err.Error(),
			Target:     hostname,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	return &EgressResponse{
		Success:     true,
		Message:     fmt.Sprintf("Successfully resolved %s to %d IP address(es)", hostname, len(ips)),
		Target:      hostname,
		ResolvedIps: ips,
		DurationMs:  duration.Milliseconds(),
	}, nil
}

// createDNSResolver creates a DNS resolver, using a custom DNS server if specified.
// Returns the system default resolver when dnsServer is empty.
// Returns an error if the dnsServer address is invalid.
func (s *InfrabinService) createDNSResolver(dnsServer string) (*net.Resolver, error) {
	normalizedDNS, err := validateDNSServerAddress(dnsServer)
	if err != nil {
		return nil, err
	}

	if normalizedDNS == "" {
		return &net.Resolver{}, nil
	}

	timeout := viper.GetDuration("egressTimeout")
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, "udp", normalizedDNS)
		},
	}, nil
}

func (s *InfrabinService) EgressHTTP(ctx context.Context, request *EgressHTTPRequest) (*EgressResponse, error) {
	return s.testHTTPConnection(ctx, request.Target, "http", 80, false)
}

func (s *InfrabinService) EgressHTTPS(ctx context.Context, request *EgressHTTPSRequest) (*EgressResponse, error) {
	return s.testHTTPConnection(ctx, request.Target, "https", 443, false)
}

func (s *InfrabinService) EgressHTTPSInsecure(ctx context.Context, request *EgressHTTPSInsecureRequest) (*EgressResponse, error) {
	return s.testHTTPConnection(ctx, request.Target, "https", 443, true)
}

// parseTargetAndDNS parses target in format "host:port@dns" or "host:port".
// Returns hostPort and optional dnsServer (empty string if not specified).
func parseTargetAndDNS(target string) (hostPort, dnsServer string) {
	// Cut at @ separator to extract host:port and optional DNS server
	hostPort, dnsServer, _ = strings.Cut(target, "@")
	return hostPort, dnsServer
}

const (
	// MaxEgressResponseBodySize is the maximum response body size to read from egress HTTP/HTTPS requests.
	// Limited to 1MB to prevent memory exhaustion from large responses.
	MaxEgressResponseBodySize = 1024 * 1024
)

// testHTTPConnection performs an HTTP/HTTPS connectivity test.
// Target format: "host:port@dns" where @dns is optional
// If DNS is specified, it will be used for resolution instead of system DNS.
//
// Design note: This function returns success/failure information in the response body
// rather than gRPC status codes. This allows clients to receive timing information
// even when the connectivity test fails, which is valuable for diagnosing network issues.
func (s *InfrabinService) testHTTPConnection(ctx context.Context, target, scheme string, defaultPort int, insecure bool) (*EgressResponse, error) {
	if target == "" {
		return nil, status.Errorf(codes.InvalidArgument, "target must not be empty")
	}

	// Check if context is already cancelled before expensive operations
	if err := ctx.Err(); err != nil {
		return &EgressResponse{
			Success: false,
			Error:   fmt.Sprintf("request cancelled: %v", err),
			Target:  target,
		}, nil
	}

	// Parse target to extract host:port and optional DNS server
	hostPort, dnsServer := parseTargetAndDNS(target)

	// Add default port if not specified
	if !strings.Contains(hostPort, ":") {
		hostPort = fmt.Sprintf("%s:%d", hostPort, defaultPort)
	}

	// Validate and normalize DNS server address if provided
	normalizedDNS, err := validateDNSServerAddress(dnsServer)
	if err != nil {
		return &EgressResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid DNS server address %q: %v", dnsServer, err),
			Target:  hostPort,
		}, nil
	}
	dnsServer = normalizedDNS

	// Build URL
	testURL := fmt.Sprintf("%s://%s", scheme, hostPort)

	// Get timeout from configuration
	timeout := viper.GetDuration("egressTimeout")

	// Create HTTP transport with appropriate TLS configuration
	// Note: We create a new transport for each request to support custom DNS servers.
	// For requests without custom DNS, connection pooling still occurs within the transport.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	// Clean up idle connections to prevent goroutine leaks
	defer transport.CloseIdleConnections()

	if dnsServer != "" {
		// Create custom resolver with specified DNS server
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: timeout,
				}
				return d.DialContext(ctx, "udp", dnsServer)
			},
		}

		// Custom dialer that uses the custom resolver
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Resolve using custom DNS
			ips, err := resolver.LookupHost(ctx, host)
			if err != nil {
				return nil, err
			}

			// Use first resolved IP
			if len(ips) == 0 {
				return nil, fmt.Errorf("no IP addresses found for %s", host)
			}

			// Dial to resolved IP
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
		}
	}

	// Create HTTP client with timeout and custom transport
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	// Make request
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return &EgressResponse{
			Success:    false,
			Error:      fmt.Sprintf("Failed to create request: %v", err),
			Target:     hostPort,
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &EgressResponse{
			Success:    false,
			Error:      err.Error(),
			Target:     hostPort,
			DurationMs: duration.Milliseconds(),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Read and discard response body to enable connection reuse within the transport.
	// Limited to MaxEgressResponseBodySize to prevent memory exhaustion.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxEgressResponseBodySize))

	return &EgressResponse{
		Success:    true,
		Message:    fmt.Sprintf("Successfully connected to %s", hostPort),
		Target:     hostPort,
		StatusCode: int32(resp.StatusCode),
		DurationMs: duration.Milliseconds(),
	}, nil
}

// SetLivenessStatus controls the liveness probe status for the "liveness" service.
// It updates the gRPC health check status that can be queried via grpc.health.v1.Health/Check.
// Accepts "pass" to mark as healthy (SERVING), "fail" to mark as unhealthy (NOT_SERVING).
func (s *InfrabinService) SetLivenessStatus(ctx context.Context, req *SetHealthStatusRequest) (*Response, error) {
	if err := validateHealthStatus(req.Status); err != nil {
		return nil, err
	}

	servingStatus := parseHealthStatus(req.Status)
	s.HealthService.SetServingStatus("liveness", servingStatus)

	return &Response{
		Liveness: fmt.Sprintf("liveness status set to %s", req.Status),
	}, nil
}

// SetReadinessStatus controls the readiness probe status for the "readiness" service.
// It updates the gRPC health check status that can be queried via grpc.health.v1.Health/Check.
// Accepts "pass" to mark as ready (SERVING), "fail" to mark as not ready (NOT_SERVING).
func (s *InfrabinService) SetReadinessStatus(ctx context.Context, req *SetHealthStatusRequest) (*Response, error) {
	if err := validateHealthStatus(req.Status); err != nil {
		return nil, err
	}

	servingStatus := parseHealthStatus(req.Status)
	s.HealthService.SetServingStatus("readiness", servingStatus)

	return &Response{
		Readiness: fmt.Sprintf("readiness status set to %s", req.Status),
	}, nil
}

// validateHealthStatus validates that the status is either "pass" or "fail".
// Returns a gRPC InvalidArgument error if the status is invalid.
func validateHealthStatus(statusValue string) error {
	switch statusValue {
	case "pass", "fail":
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be 'pass' or 'fail', got: %s", statusValue)
	}
}

// parseHealthStatus converts a string status to the gRPC health status constant.
// This function assumes the status has already been validated by validateHealthStatus.
// "pass" returns SERVING, "fail" returns NOT_SERVING, any other value defaults to SERVING.
func parseHealthStatus(statusValue string) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch statusValue {
	case "fail":
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		return grpc_health_v1.HealthCheckResponse_SERVING
	}
}
