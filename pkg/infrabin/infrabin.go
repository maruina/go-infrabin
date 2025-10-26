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
	intermittentErrorsCounter atomic.Int32
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
	time.Sleep(duration)

	return &Response{Delay: int32(duration.Seconds())}, nil
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

	// Send http request
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach %s: %v", request.Url, err)
	}

	// Read request body and close it
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error reading upstream response body: %v", err)
	}
	if err = resp.Body.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "Error closing upstream response: %v", err)
	}

	// Convert []bytes into json struct
	var response structpb.Struct
	if err := response.UnmarshalJSON(body); err != nil {
		return nil, status.Errorf(codes.Internal, "Error creating Struct from upstream response json: %v", err)
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
		return nil, status.Errorf(codes.Internal, "Error assuming AWS IAM role, %v", err)
	}

	return &Response{AssumedRoleId: roleId}, nil
}

func (s *InfrabinService) AWSGetCallerIdentity(ctx context.Context, _ *Empty) (*Response, error) {
	response, err := aws.STSGetCallerIdentity(ctx, s.STSClient)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error calling AWS Get Caller Identity, %v", err)
	}

	return &Response{
		GetCallerIdentity: &GetCallerIdentityResponse{
			Account: *response.Account,
			Arn:     *response.Arn,
			UserId:  *response.UserId,
		},
	}, nil
}

func (s *InfrabinService) Intermittent(ctx context.Context, _ *Empty) (*Response, error) {
	maxErrs := viper.GetInt32("intermittentErrors")

	counter := s.intermittentErrorsCounter.Add(1)
	if counter <= maxErrs {
		return nil, status.Errorf(codes.Unavailable, "%d errors left", maxErrs-counter+1)
	}

	s.intermittentErrorsCounter.Store(0)
	return &Response{
		Intermittent: &IntermittentResponse{
			IntermittentErrors: maxErrs,
		},
	}, nil
}

func (s *InfrabinService) RandomData(ctx context.Context, request *RandomDataRequest) (*Response, error) {
	response, err := random.Bytes(int(request.GetPath()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error generating data, %v", err)
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
// Returns system resolver when dnsServer is empty.
func (s *InfrabinService) createDNSResolver(dnsServer string) (*net.Resolver, error) {
	if dnsServer == "" {
		return &net.Resolver{}, nil
	}

	// Add default port 53 if not specified
	if !strings.Contains(dnsServer, ":") {
		dnsServer = fmt.Sprintf("%s:53", dnsServer)
	}

	// Validate DNS server address format
	if _, _, err := net.SplitHostPort(dnsServer); err != nil {
		return nil, fmt.Errorf("invalid DNS server address %q: %w", dnsServer, err)
	}

	timeout := viper.GetDuration("egressTimeout")
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, "udp", dnsServer)
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

// parseTargetAndDNS parses target in format "host:port@dns" or "host:port"
// Returns hostPort, dnsServer, and error
func parseTargetAndDNS(target string) (hostPort, dnsServer string) {
	// Split by @ to separate host:port from DNS server
	parts := strings.Split(target, "@")
	hostPort = parts[0]
	if len(parts) > 1 {
		dnsServer = parts[1]
	}
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

	// Parse target to extract host:port and optional DNS server
	hostPort, dnsServer := parseTargetAndDNS(target)

	// Add default port if not specified
	if !strings.Contains(hostPort, ":") {
		hostPort = fmt.Sprintf("%s:%d", hostPort, defaultPort)
	}

	// Validate DNS server address if provided
	if dnsServer != "" {
		if _, _, err := net.SplitHostPort(dnsServer); err != nil {
			return &EgressResponse{
				Success: false,
				Error:   fmt.Sprintf("invalid DNS server address %q: %v", dnsServer, err),
				Target:  hostPort,
			}, nil
		}
	}

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
