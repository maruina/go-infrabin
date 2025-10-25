package infrabin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infrabin_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "handler", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "infrabin_http_request_duration_seconds",
			Help:    "HTTP request latency distributions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "handler"},
	)
)

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

// HTTPMetricsMiddleware instruments HTTP requests with Prometheus metrics
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mrw := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(mrw, r)

		duration := time.Since(start).Seconds()
		route := normalizeRoute(r.URL.Path)
		status := strconv.Itoa(mrw.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(duration)
	})
}

// normalizeRoute extracts the handler name from the path to prevent cardinality explosion
//
// This function returns the endpoint name similar to gRPC method names, ensuring consistent
// metric labeling across both HTTP and gRPC layers. Dynamic path segments are ignored.
//
// MAINTENANCE: When adding new endpoints to proto/infrabin/infrabin.proto, update this function
// to map the endpoint path to its handler name.
//
// Examples:
//   - /delay/5 -> delay
//   - /env/HOME -> env
//   - /aws/metadata/instance-id -> aws-metadata
//   - / -> root
func normalizeRoute(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Handle root path
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return "root"
	}

	// Extract handler name from path
	if len(parts) >= 1 {
		endpoint := parts[0]
		switch endpoint {
		case "delay":
			return "delay"
		case "headers":
			return "headers"
		case "env":
			return "env"
		case "proxy":
			return "proxy"
		case "intermittent":
			return "intermittent"
		case "any":
			return "any"
		case "bytes":
			return "bytes"
		case "aws":
			// Handle AWS sub-paths
			if len(parts) >= 2 {
				switch parts[1] {
				case "metadata":
					return "aws-metadata"
				case "assume":
					return "aws-assume"
				case "get-caller-identity":
					return "aws-get-caller-identity"
				}
			}
			return "aws"
		}
	}

	// Default: return the first path segment for unknown patterns
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "unknown"
}
