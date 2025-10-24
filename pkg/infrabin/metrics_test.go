package infrabin

import (
	"testing"
)

func TestNormalizeRoute(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		expectedRoute string
	}{
		{
			name:          "root path",
			path:          "/",
			expectedRoute: "/",
		},
		{
			name:          "headers endpoint",
			path:          "/headers",
			expectedRoute: "/headers",
		},
		{
			name:          "delay with parameter",
			path:          "/delay/5",
			expectedRoute: "/delay/{duration}",
		},
		{
			name:          "env with parameter",
			path:          "/env/HOME",
			expectedRoute: "/env/{env_var}",
		},
		{
			name:          "aws metadata with path",
			path:          "/aws/metadata/instance-id",
			expectedRoute: "/aws/metadata/{path}",
		},
		{
			name:          "aws assume with role",
			path:          "/aws/assume/arn:aws:iam::123456789012:role/MyRole",
			expectedRoute: "/aws/assume/{role}",
		},
		{
			name:          "aws get-caller-identity",
			path:          "/aws/get-caller-identity",
			expectedRoute: "/aws/get-caller-identity",
		},
		{
			name:          "any with path",
			path:          "/any/foo/bar/baz",
			expectedRoute: "/any/{path}",
		},
		{
			name:          "bytes with path",
			path:          "/bytes/1024",
			expectedRoute: "/bytes/{path}",
		},
		{
			name:          "proxy endpoint",
			path:          "/proxy",
			expectedRoute: "/proxy",
		},
		{
			name:          "intermittent endpoint",
			path:          "/intermittent",
			expectedRoute: "/intermittent",
		},
		{
			name:          "unknown endpoint returns as-is",
			path:          "/unknown/path",
			expectedRoute: "/unknown/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeRoute(tc.path)
			if result != tc.expectedRoute {
				t.Errorf("normalizeRoute(%q) = %q, want %q", tc.path, result, tc.expectedRoute)
			}
		})
	}
}
