package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maruina/go-infrabin/pkg/infrabin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start go-infrabin",
	Long: `Start go-infrabin to serve traffic.

By default the proxy endpoints are disabled.`,
	Run: func(cmd *cobra.Command, args []string) {

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
			"http",
			infrabin.RegisterInfrabin("/", grpcServer.InfrabinService),
		)
		go server.ListenAndServe()

		// run admin server in background
		admin := infrabin.NewHTTPServer(
			"admin",
			infrabin.RegisterHealth("/healthcheck/liveness/", grpcServer.HealthService),
			infrabin.RegisterHealth("/healthcheck/readiness/", grpcServer.HealthService),
		)
		go admin.ListenAndServe()

		// run Prometheus server
		promServer := infrabin.NewHTTPServer(
			"prom",
			infrabin.RegisterHandler("/", promhttp.Handler()),
		)
		go promServer.ListenAndServe()

		// wait for SIGINT
		<-finish

		admin.Shutdown()
		server.Shutdown()
		grpcServer.Shutdown()
		promServer.Shutdown()

	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Bool("enable-proxy-endpoint", false, "If true, enables proxy and aws endpoints.")
	serveCmd.Flags().String("grpc-host", "0.0.0.0", "the address to bind the gRPC server.")
	serveCmd.Flags().Int("grpc-port", 50051, "the gRPC server port.")
	serveCmd.Flags().String("http-host", "0.0.0.0", "the address to bind the HTTP server.")
	serveCmd.Flags().Int("http-port", 8888, "the HTTP server port.")
	serveCmd.Flags().String("admin-host", "0.0.0.0", "the address to bind the admin server.")
	serveCmd.Flags().Int("admin-port", 8889, "the admin server port.")
	serveCmd.Flags().String("prom-host", "0.0.0.0", "the address to bind the Prometheus metric server.")
	serveCmd.Flags().Int("prom-port", 8887, "the Prometheus server port.")
	serveCmd.Flags().String("aws-metadata-endpoint", "http://169.254.169.254/latest/meta-data/", "the AWS metadata endpoint URL.")
	serveCmd.Flags().Duration("max-delay-endpoint", 120*time.Second, "the max delay for the delay endpoint.")
	serveCmd.Flags().Duration("drain-timeout", 120*time.Second, "the drain timeout to allow terminating in-flight requests.")
	serveCmd.Flags().Duration("http-write-timeout", 120*time.Second, "the maximum duration before timing out writes of the response.")
	serveCmd.Flags().Duration("http-read-timeout", 60*time.Second, "maximum duration for reading the entire request, including the body.")
	serveCmd.Flags().Duration("http-idle-timeout", 15*time.Second, "the maximum amount of time to wait for the next request when keep-alives are enabled.")
	serveCmd.Flags().Duration("http-read-header-timeout", 15*time.Second, "the amount of time allowed to read request headers.")

	// Bind every cobra flag to viper
	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
