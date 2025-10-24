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

// normalizeRoute converts dynamic path segments to patterns to prevent cardinality explosion
//
// MAINTENANCE: When adding new endpoints to proto/infrabin/infrabin.proto, update this function
// to map dynamic path segments to their pattern equivalents. This ensures metric cardinality
// remains bounded regardless of parameter values.
//
// Examples:
//   - /delay/5 -> /delay/{duration}
//   - /env/HOME -> /env/{env_var}
//   - /aws/metadata/instance-id -> /aws/metadata/{path}
func normalizeRoute(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Handle root path
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return "/"
	}

	// Known endpoint patterns - prevents cardinality explosion
	// by mapping dynamic segments like /delay/5 -> /delay/{duration}
	if len(parts) >= 1 {
		endpoint := parts[0]
		switch endpoint {
		case "delay":
			return "/delay/{duration}"
		case "headers":
			return "/headers"
		case "env":
			return "/env/{env_var}"
		case "proxy":
			return "/proxy"
		case "intermittent":
			return "/intermittent"
		case "any":
			return "/any/{path}"
		case "bytes":
			return "/bytes/{path}"
		case "aws":
			// Handle AWS sub-paths
			if len(parts) >= 2 {
				switch parts[1] {
				case "metadata":
					return "/aws/metadata/{path}"
				case "assume":
					return "/aws/assume/{role}"
				case "get-caller-identity":
					return "/aws/get-caller-identity"
				}
			}
			return "/aws"
		}
	}

	// Default: return the path as-is for unknown patterns
	return path
}
