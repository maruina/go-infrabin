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
			for vFlag, cFlag := range map[string]string{
				"grpc.host":             "grpc-host",
				"grpc.port":             "grpc-port",
				"server.host":           "server-host",
				"server.port":           "server-port",
				"admin.host":            "admin-host",
				"admin.port":            "admin-port",
				"prom.host":             "prom-host",
				"prom.port":             "prom-port",
				"proxyEndpoint":         "enable-proxy-endpoint",
				"awsMetadataEndpoint":   "aws-metadata-endpoint",
				"drainTimeout":          "drain-timeout",
				"maxDelay":              "max-delay",
				"httpWriteTimeout":      "http-write-timeout",
				"httpReadTimeout":       "http-read-timeout",
				"httpIdleTimeout":       "http-idle-timeout",
				"httpReadHeaderTimeout": "http-read-header-timeout",
			} {
				if err := viper.BindPFlag(vFlag, cmd.Flags().Lookup(cFlag)); err != nil {
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
	cobra.OnInitialize(infrabin.ReadConfiguration)

	rootCmd.Flags().IP("grpc-host", net.ParseIP(infrabin.DefaultHost), "gRPC host")
	rootCmd.Flags().Uint("grpc-port", infrabin.DefaultGRPCPort, "gRPC port")
	rootCmd.Flags().IP("server-host", net.ParseIP(infrabin.DefaultHost), "HTTP server host")
	rootCmd.Flags().Uint("server-port", infrabin.DefaultHTTPServerPort, "HTTP server port")
	rootCmd.Flags().IP("admin-host", net.ParseIP(infrabin.DefaultHost), "HTTP server host")
	rootCmd.Flags().IP("prom-host", net.ParseIP(infrabin.DefaultHost), "Prometheus metrics host")
	rootCmd.Flags().Uint("prom-port", infrabin.DefaultPrometheusPort, "Prometheus metrics port")
	rootCmd.Flags().Bool("enable-proxy-endpoint", infrabin.EnableProxyEndpoint, "When enabled allows /proxy and /aws endpoints")
	rootCmd.Flags().String("aws-metadata-endpoint", infrabin.AWSMetadataEndpoint, "AWS Metadata Endpoint")
	rootCmd.Flags().Duration("drain-timeout", infrabin.DrainTimeout, "Drain timeout")
	rootCmd.Flags().Duration("max-delay", infrabin.MaxDelay, "Maximum delay")
	rootCmd.Flags().Duration("http-write-timeout", infrabin.HttpWriteTimeout, "HTTP write timeout")
	rootCmd.Flags().Duration("http-read-timeout", infrabin.HttpReadTimeout, "HTTP read timeout")
	rootCmd.Flags().Duration("http-idle-timeout", infrabin.HttpIdleTimeout, "HTTP idle timeout")
	rootCmd.Flags().Duration("http-read-header-timeout", infrabin.HttpReadHeaderTimeout, "HTTP read header timeout")
}

func run(cmd *cobra.Command, args []string) {
	// Create a channel to catch signals
	finish := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// and SIGTERM (used in docker and kubernetes)
	// SIGKILL or SIGQUIT will not be caught.
	signal.Notify(finish, syscall.SIGINT, syscall.SIGTERM)

	// run grpc server in background
	grpcServer := infrabin.NewGRPCServer()
	go grpcServer.ListenAndServe(nil)

	// run service server in background
	server := infrabin.NewHTTPServer(
		"server",
		infrabin.RegisterInfrabin("/", grpcServer.InfrabinService),
	)
	go server.ListenAndServe()

	// run Prometheus server
	promServer := infrabin.NewHTTPServer(
		"prom",
		infrabin.RegisterHandler("/", promhttp.Handler()),
	)
	go promServer.ListenAndServe()

	// wait for SIGINT
	<-finish

	server.Shutdown()
	grpcServer.Shutdown()
	promServer.Shutdown()
}
