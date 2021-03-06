package infrabin

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	AppName                    = "go-infrabin"
	DefaultHost                = "0.0.0.0"
	DefaultGRPCPort       uint = 50051
	DefaultHTTPServerPort uint = 8888
	DefaultPrometheusPort uint = 8887
	EnableProxyEndpoint        = false
	AWSMetadataEndpoint        = "http://169.254.169.254/latest/meta-data/"
	DrainTimeout               = 15 * time.Second
	MaxDelay                   = 120 * time.Second
	HttpWriteTimeout           = MaxDelay + time.Second
	HttpReadTimeout            = 60 * time.Second
	HttpIdleTimeout            = 15 * time.Second
	HttpReadHeaderTimeout      = 15 * time.Second
	DefaultConfigName          = "config"
	DefaultConfigType          = "yaml"
)

var (
	DefaultConfigPaths = [...]string{".", "./config", "./cmd/go-infrabin"}
)

// ReadConfiguration : Sets up a default configuration then overwrites any configured details from
// a config.yaml file in the local directory or under ./config
func ReadConfiguration() {

	// Config file should be config.yaml
	viper.SetConfigName(DefaultConfigName)
	viper.SetConfigType(DefaultConfigType)

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

	// Other Infrastructure Defaults
	viper.SetDefault("awsMetadataEndpoint", AWSMetadataEndpoint)

	// Graceful timeout duration
	viper.SetDefault("drainTimeout", DrainTimeout)

	// Max delay duration for Delay endpoint
	viper.SetDefault("maxDelay", MaxDelay)

	// http timeouts
	viper.SetDefault("httpWriteTimeout", HttpWriteTimeout)
	viper.SetDefault("httpReadTimeout", HttpReadTimeout)
	viper.SetDefault("httpIdleTimeout", HttpIdleTimeout)
	viper.SetDefault("httpReadHeaderTimeout", HttpReadHeaderTimeout)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Will just use the default configuration.
		} else {
			// Config file was found but another error was produced, so we will exit.
			panic(fmt.Errorf("Fatal error config file: %s", err))
		}
	}
}
