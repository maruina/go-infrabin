package infrabin

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/handlers"
)

// RequestLoggingFormatter extends Apache Combined Log Format with proper sourceIP handling
// Format: sourceIP - remoteUser [timestamp] "method path protocol" statusCode size "referer" "userAgent"
func RequestLoggingFormatter(writer io.Writer, params handlers.LogFormatterParams) {
	sourceIP := getSourceIP(params.Request)

	// Get remote user from Basic Auth, default to "-"
	username, _, _ := params.Request.BasicAuth()
	if username == "" {
		username = "-"
	}

	// Get referer, default to "-"
	referer := params.Request.Referer()
	if referer == "" {
		referer = "-"
	}

	// Get user agent, default to "-"
	userAgent := params.Request.UserAgent()
	if userAgent == "" {
		userAgent = "-"
	}

	// Apache Combined Log Format with sourceIP instead of RemoteAddr
	fmt.Fprintf(writer, "%s - %s [%s] \"%s %s %s\" %d %d \"%s\" \"%s\"\n",
		sourceIP,
		username,
		params.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
		params.Request.Method,
		params.URL.RequestURI(),
		params.Request.Proto,
		params.StatusCode,
		params.Size,
		referer,
		userAgent,
	)
}

// getSourceIP extracts the source IP from the request, considering proxy headers
func getSourceIP(r *http.Request) string {
	// Check X-Real-IP header first
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Check X-Forwarded-For header (may contain multiple IPs, take the first)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// Split by comma and take the first IP
		if idx := strings.Index(forwardedFor, ","); idx != -1 {
			return strings.TrimSpace(forwardedFor[:idx])
		}
		return forwardedFor
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}
