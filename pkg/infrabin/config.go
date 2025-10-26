package infrabin

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	AppName                    = "go-infrabin"
	AWSMetadataEndpoint        = "http://169.254.169.254/latest/meta-data/"
	DefaultConfigName          = "config"
	DefaultConfigType          = "yaml"
	DefaultGRPCPort       uint = 50051
	DefaultHost                = "0.0.0.0"
	DefaultHTTPServerPort uint = 8888
	DefaultPrometheusPort uint = 8887
	DrainTimeout               = 15 * time.Second
	EnableProxyEndpoint        = false
	HTTPIdleTimeout            = 15 * time.Second
	HTTPReadHeaderTimeout      = 15 * time.Second
	HTTPReadTimeout            = 60 * time.Second
	HTTPWriteTimeout           = MaxDelay + time.Second
	MaxDelay                   = 120 * time.Second
	ProxyAllowRegexp           = ".*"
	IntermittentErrors         = 2
	// EgressTimeout is the default timeout for egress HTTP/HTTPS connectivity tests.
	// Set to 3 seconds to balance between detecting connection issues quickly and
	// allowing sufficient time for slower networks or high-latency connections.
	EgressTimeout = 3 * time.Second
)

var (
	DefaultConfigPaths = [...]string{".", "./config", "./cmd/go-infrabin"}
)

// ReadConfiguration sets up a default configuration then overwrites any configured details from
// a config.yaml file in the local directory or under ./config
func ReadConfiguration() error {

	// Config file should be config.yaml
	viper.SetConfigName(DefaultConfigName)
	viper.SetConfigType(DefaultConfigType)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Default locations should be in the current directory or in a sub-directory called config.
	// Future iterations should take a config file from an environmental variable via spf13/cobra
	for _, path := range DefaultConfigPaths {
		viper.AddConfigPath(path)
	}

	// This is where we set the overwritable defaults prior to attempting the parse of the configuration.

	// gRPC Defaults
	viper.SetDefault("grpc.host", DefaultHost)
	viper.SetDefault("grpc.port", DefaultGRPCPort)

	// http server Defaults
	viper.SetDefault("server.host", DefaultHost)
	viper.SetDefault("server.port", DefaultHTTPServerPort)

	// Prometheus Defaults
	viper.SetDefault("prom.host", DefaultHost)
	viper.SetDefault("prom.port", DefaultPrometheusPort)

	// ProxyEndpoint Configuration
	viper.SetDefault("proxyEndpoint", EnableProxyEndpoint)
	viper.SetDefault("proxyAllowRegexp", ProxyAllowRegexp)

	// Other Infrastructure Defaults
	viper.SetDefault("awsMetadataEndpoint", AWSMetadataEndpoint)

	// Graceful timeout duration
	viper.SetDefault("drainTimeout", DrainTimeout)

	// Max delay duration for Delay endpoint
	viper.SetDefault("maxDelay", MaxDelay)

	// Consecutive errors for intermittent endpoint
	viper.SetDefault("intermittentErrors", IntermittentErrors)

	// Egress endpoint timeout
	viper.SetDefault("egressTimeout", EgressTimeout)

	// http timeouts
	viper.SetDefault("httpWriteTimeout", HTTPWriteTimeout)
	viper.SetDefault("httpReadTimeout", HTTPReadTimeout)
	viper.SetDefault("httpIdleTimeout", HTTPIdleTimeout)
	viper.SetDefault("httpReadHeaderTimeout", HTTPReadHeaderTimeout)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Will just use the default configuration.
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("fatal error config file: %w", err)
		}
	}
	return nil
}
