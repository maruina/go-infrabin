// Package cmd provides the command-line interface and initialization logic
// for the go-infrabin HTTP and gRPC servers.
package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/maruina/go-infrabin/pkg/infrabin"
)

var (
	rootCmd = &cobra.Command{
		Use:  infrabin.AppName,
		Long: fmt.Sprintf("%s is an HTTP and GRPC server that can be used to simulate blue/green deployments, to test routing and failover or as a general swiss-knife for your infrastructure.", infrabin.AppName),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			for viperKey, cobraFlag := range map[string]string{
				"grpc.host":             "grpc-host",
				"grpc.port":             "grpc-port",
				"server.host":           "server-host",
				"server.port":           "server-port",
				"prom.host":             "prom-host",
				"prom.port":             "prom-port",
				"proxyEndpoint":         "enable-proxy-endpoint",
				"proxyAllowRegexp":      "proxy-allow-regexp",
				"awsMetadataEndpoint":   "aws-metadata-endpoint",
				"drainTimeout":          "drain-timeout",
				"maxDelay":              "max-delay",
				"httpWriteTimeout":      "http-write-timeout",
				"httpReadTimeout":       "http-read-timeout",
				"httpIdleTimeout":       "http-idle-timeout",
				"httpReadHeaderTimeout": "http-read-header-timeout",
				"intermittentErrors":    "intermittent-errors",
			} {
				if err := viper.BindPFlag(viperKey, cmd.Flags().Lookup(cobraFlag)); err != nil {
					return err
				}
			}
			return nil
		},
		Run: run,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Parse the configuration or set default configuration
	cobra.OnInitialize(func() {
		if err := infrabin.ReadConfiguration(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading configuration: %v\n", err)
			os.Exit(1)
		}
	})

	rootCmd.Flags().IP("grpc-host", net.ParseIP(infrabin.DefaultHost), "gRPC host")
	rootCmd.Flags().Uint("grpc-port", infrabin.DefaultGRPCPort, "gRPC port")
	rootCmd.Flags().IP("server-host", net.ParseIP(infrabin.DefaultHost), "HTTP server host")
	rootCmd.Flags().Uint("server-port", infrabin.DefaultHTTPServerPort, "HTTP server port")
	rootCmd.Flags().IP("prom-host", net.ParseIP(infrabin.DefaultHost), "Prometheus metrics host")
	rootCmd.Flags().Uint("prom-port", infrabin.DefaultPrometheusPort, "Prometheus metrics port")
	rootCmd.Flags().Bool("enable-proxy-endpoint", infrabin.EnableProxyEndpoint, "When enabled allows /proxy and /aws endpoints")
	rootCmd.Flags().String("proxy-allow-regexp", infrabin.ProxyAllowRegexp, "Regexp to allow URLs via /proxy endpoint")
	rootCmd.Flags().String("aws-metadata-endpoint", infrabin.AWSMetadataEndpoint, "AWS Metadata Endpoint")
	rootCmd.Flags().Duration("drain-timeout", infrabin.DrainTimeout, "Drain timeout")
	rootCmd.Flags().Duration("max-delay", infrabin.MaxDelay, "Maximum delay")
	rootCmd.Flags().Duration("http-write-timeout", infrabin.HTTPWriteTimeout, "HTTP write timeout")
	rootCmd.Flags().Duration("http-read-timeout", infrabin.HTTPReadTimeout, "HTTP read timeout")
	rootCmd.Flags().Duration("http-idle-timeout", infrabin.HTTPIdleTimeout, "HTTP idle timeout")
	rootCmd.Flags().Duration("http-read-header-timeout", infrabin.HTTPReadHeaderTimeout, "HTTP read header timeout")
	rootCmd.Flags().Int32("intermittent-errors", infrabin.IntermittentErrors, "Consecutive 503 errors before returning 200 for the /intermittent endpoint")
}

func run(cmd *cobra.Command, args []string) {
	// Create a channel to catch signals
	finish := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// and SIGTERM (used in docker and kubernetes)
	// SIGKILL or SIGQUIT will not be caught.
	signal.Notify(finish, syscall.SIGINT, syscall.SIGTERM)

	// run grpc server in background
	grpcServer, err := infrabin.NewGRPCServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize gRPC server: %v\n", err)
		os.Exit(1)
	}
	go grpcServer.ListenAndServe(nil)

	// run service server in background
	server, err := infrabin.NewHTTPServer(
		"server",
		infrabin.RegisterInfrabin("/", grpcServer.InfrabinService),
		infrabin.RegisterOpenAPI("/openapi.json"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize HTTP server: %v\n", err)
		os.Exit(1)
	}
	go server.ListenAndServe()

	// run Prometheus server
	promServer, err := infrabin.NewHTTPServer(
		"prom",
		infrabin.RegisterHandler("/", promhttp.Handler()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Prometheus server: %v\n", err)
		os.Exit(1)
	}
	go promServer.ListenAndServe()

	// wait for SIGINT
	<-finish

	server.Shutdown()
	grpcServer.Shutdown()
	promServer.Shutdown()
}
