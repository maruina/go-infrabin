package infrabin

import (
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultConfig(t *testing.T) {

	ReadConfiguration()

	var tests = []struct {
		name  string
		value string
	}{
		{"grpc.host", "0.0.0.0"},
		{"grpc.port", "50051"},

		{"server.host", "0.0.0.0"},
		{"server.port", "8888"},

		{"admin.host", "0.0.0.0"},
		{"admin.port", "8889"},

		{"prom.host", "0.0.0.0"},
		{"prom.port", "8887"},

		{"proxyEndpoint", "false"},
		{"awsMetadataEndpoint", "http://169.254.169.254/latest/meta-data/"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			ans := viper.GetString(tt.name)
			if ans != tt.value {
				t.Errorf("Got %s, wanted %s", ans, tt.value)
			}
		})
	}
}
