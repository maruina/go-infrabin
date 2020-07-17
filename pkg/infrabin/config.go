package infrabin

import (
	"fmt"

	"github.com/spf13/viper"
)

// ReadConfiguration : Sets up a default configuration then overwrites any configured details from
// a config.yaml file in the local directory or under ./config
func ReadConfiguration() {

	// Config file should be config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Default locations should be in the current directory or in a sub-directory called config.
	// Future iterations should take a config file from an environmental variable via spf13/cobra
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./cmd/go-infrabin")

	// This is where we set the overwritable defaults prior to attempting the parse of the configuration.

	// gRPC Defaults
	viper.SetDefault("grpc.host", "0.0.0.0")
	viper.SetDefault("grpc.port", 50051)

	// http server Defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8888)

	// admin Defaults
	viper.SetDefault("admin.host", "0.0.0.0")
	viper.SetDefault("admin.port", 8889)

	// Prometheus Defaults
	viper.SetDefault("prom.host", "0.0.0.0")
	viper.SetDefault("prom.port", 8887)

	// ProxyEndpoint Configuration
	viper.SetDefault("proxyEndpoint", false)

	// Other Infrastructure Defaults
	viper.SetDefault("awsMetadataEndpoint", "http://169.254.169.254/latest/meta-data/")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Will just use the default configuration.
		} else {
			// Config file was found but another error was produced, so we will exit.
			panic(fmt.Errorf("Fatal error config file: %s", err))
		}
	}
}
