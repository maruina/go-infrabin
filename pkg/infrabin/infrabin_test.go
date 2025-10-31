package infrabin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// mockHealthService is a mock implementation of HealthService for testing.
type mockHealthService struct {
	lastService string
	lastStatus  grpc_health_v1.HealthCheckResponse_ServingStatus
}

func (m *mockHealthService) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	m.lastService = service
	m.lastStatus = status
}

func TestSetLivenessStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		wantService    string
		wantStatus     grpc_health_v1.HealthCheckResponse_ServingStatus
		wantErr        bool
		wantErrCode    codes.Code
		wantErrMessage string
	}{
		{
			name:        "pass sets to SERVING",
			status:      "pass",
			wantService: "liveness",
			wantStatus:  grpc_health_v1.HealthCheckResponse_SERVING,
			wantErr:     false,
		},
		{
			name:        "fail sets to NOT_SERVING",
			status:      "fail",
			wantService: "liveness",
			wantStatus:  grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			wantErr:     false,
		},
		{
			name:           "invalid status returns error",
			status:         "invalid",
			wantErr:        true,
			wantErrCode:    codes.InvalidArgument,
			wantErrMessage: "status must be 'pass' or 'fail'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := &mockHealthService{}
			service := &InfrabinService{
				HealthService: mock,
			}

			req := &SetHealthStatusRequest{Status: tt.status}
			resp, err := service.SetLivenessStatus(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetLivenessStatus() expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("SetLivenessStatus() error is not a status error: %v", err)
				}
				if st.Code() != tt.wantErrCode {
					t.Errorf("SetLivenessStatus() error code = %v, want %v", st.Code(), tt.wantErrCode)
				}
				if tt.wantErrMessage != "" && !strings.Contains(st.Message(), tt.wantErrMessage) {
					t.Errorf("SetLivenessStatus() error message = %v, want to contain %v", st.Message(), tt.wantErrMessage)
				}
				return
			}

			if err != nil {
				t.Fatalf("SetLivenessStatus() returned unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatalf("SetLivenessStatus() returned nil response")
			}

			if mock.lastService != tt.wantService {
				t.Errorf("SetLivenessStatus() service = %v, want %v", mock.lastService, tt.wantService)
			}

			if mock.lastStatus != tt.wantStatus {
				t.Errorf("SetLivenessStatus() status = %v, want %v", mock.lastStatus, tt.wantStatus)
			}
		})
	}
}

func TestSetReadinessStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		wantService    string
		wantStatus     grpc_health_v1.HealthCheckResponse_ServingStatus
		wantErr        bool
		wantErrCode    codes.Code
		wantErrMessage string
	}{
		{
			name:        "pass sets to SERVING",
			status:      "pass",
			wantService: "readiness",
			wantStatus:  grpc_health_v1.HealthCheckResponse_SERVING,
			wantErr:     false,
		},
		{
			name:        "fail sets to NOT_SERVING",
			status:      "fail",
			wantService: "readiness",
			wantStatus:  grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			wantErr:     false,
		},
		{
			name:           "invalid status returns error",
			status:         "invalid",
			wantErr:        true,
			wantErrCode:    codes.InvalidArgument,
			wantErrMessage: "status must be 'pass' or 'fail'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := &mockHealthService{}
			service := &InfrabinService{
				HealthService: mock,
			}

			req := &SetHealthStatusRequest{Status: tt.status}
			resp, err := service.SetReadinessStatus(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetReadinessStatus() expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("SetReadinessStatus() error is not a status error: %v", err)
				}
				if st.Code() != tt.wantErrCode {
					t.Errorf("SetReadinessStatus() error code = %v, want %v", st.Code(), tt.wantErrCode)
				}
				if tt.wantErrMessage != "" && !strings.Contains(st.Message(), tt.wantErrMessage) {
					t.Errorf("SetReadinessStatus() error message = %v, want to contain %v", st.Message(), tt.wantErrMessage)
				}
				return
			}

			if err != nil {
				t.Fatalf("SetReadinessStatus() returned unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatalf("SetReadinessStatus() returned nil response")
			}

			if mock.lastService != tt.wantService {
				t.Errorf("SetReadinessStatus() service = %v, want %v", mock.lastService, tt.wantService)
			}

			if mock.lastStatus != tt.wantStatus {
				t.Errorf("SetReadinessStatus() status = %v, want %v", mock.lastStatus, tt.wantStatus)
			}
		})
	}
}

func TestParseHealthStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantStatus grpc_health_v1.HealthCheckResponse_ServingStatus
	}{
		{
			name:       "pass returns SERVING",
			status:     "pass",
			wantStatus: grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name:       "fail returns NOT_SERVING",
			status:     "fail",
			wantStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		},
		{
			name:       "any other value returns SERVING",
			status:     "unknown",
			wantStatus: grpc_health_v1.HealthCheckResponse_SERVING,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseHealthStatus(tt.status)
			if got != tt.wantStatus {
				t.Errorf("parseHealthStatus() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name string
		ptr  *string
		want string
	}{
		{
			name: "nil pointer returns empty string",
			ptr:  nil,
			want: "",
		},
		{
			name: "non-nil pointer returns value",
			ptr:  stringPtr("test"),
			want: "test",
		},
		{
			name: "empty string value returns empty string",
			ptr:  stringPtr(""),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stringValue(tt.ptr)
			if got != tt.want {
				t.Errorf("stringValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHealthStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		wantErr        bool
		wantErrCode    codes.Code
		wantErrMessage string
	}{
		{
			name:    "pass is valid",
			status:  "pass",
			wantErr: false,
		},
		{
			name:    "fail is valid",
			status:  "fail",
			wantErr: false,
		},
		{
			name:           "invalid status returns error",
			status:         "invalid",
			wantErr:        true,
			wantErrCode:    codes.InvalidArgument,
			wantErrMessage: "status must be 'pass' or 'fail'",
		},
		{
			name:           "empty status returns error",
			status:         "",
			wantErr:        true,
			wantErrCode:    codes.InvalidArgument,
			wantErrMessage: "status must be 'pass' or 'fail'",
		},
		{
			name:           "case sensitive - Pass is invalid",
			status:         "Pass",
			wantErr:        true,
			wantErrCode:    codes.InvalidArgument,
			wantErrMessage: "status must be 'pass' or 'fail'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateHealthStatus(tt.status)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateHealthStatus() expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("validateHealthStatus() error is not a status error: %v", err)
				}
				if st.Code() != tt.wantErrCode {
					t.Errorf("validateHealthStatus() error code = %v, want %v", st.Code(), tt.wantErrCode)
				}
				if tt.wantErrMessage != "" && !strings.Contains(st.Message(), tt.wantErrMessage) {
					t.Errorf("validateHealthStatus() error message = %v, want to contain %v", st.Message(), tt.wantErrMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("validateHealthStatus() unexpected error: %v", err)
			}
		})
	}
}

func TestDelayContextCancellation(t *testing.T) {
	// Set up viper configuration for maxDelay
	viper.Set("maxDelay", 10*time.Second)
	defer viper.Reset()

	service := &InfrabinService{}

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Request a delay longer than the context timeout
	req := &DelayRequest{Duration: 5}

	start := time.Now()
	resp, err := service.Delay(ctx, req)
	elapsed := time.Since(start)

	// Should return error due to context cancellation
	if err == nil {
		t.Fatalf("Delay() expected error due to context cancellation, got response: %v", resp)
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Delay() error is not a status error: %v", err)
	}

	// Should be DeadlineExceeded or Canceled
	if st.Code() != codes.DeadlineExceeded && st.Code() != codes.Canceled {
		t.Errorf("Delay() error code = %v, want DeadlineExceeded or Canceled", st.Code())
	}

	// Should complete quickly (not wait for the full 5 seconds)
	if elapsed > 1*time.Second {
		t.Errorf("Delay() took too long to cancel: %v, should be less than 1s", elapsed)
	}
}

// stringPtr is a helper function to create string pointers for tests
func stringPtr(s string) *string {
	return &s
}
