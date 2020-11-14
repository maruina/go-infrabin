package cmd

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/maruina/go-infrabin/pkg/infrabin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Create a channel to catch signals
		finish := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// and SIGTERM (used in docker and kubernetes)
		// SIGKILL or SIGQUIT will not be caught.
		signal.Notify(finish, syscall.SIGINT, syscall.SIGTERM)

		// Parse the configuration or set default configuration
		infrabin.ReadConfiguration()

		proxyEndpoint := viper.GetBool("proxyEndpoint")

		flag.BoolVar(
			&proxyEndpoint,
			"enable-proxy-endpoint",
			false,
			"If true, enables proxy and aws endpoints",
		)
		flag.Parse()

		// run grpc server in background
		grpcServer := infrabin.NewGRPCServer()
		go grpcServer.ListenAndServe(nil)

		// run service server in background
		server := infrabin.NewHTTPServer(
			"server",
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
