package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/maruina/go-infrabin/pkg/infrabin"
)

func main() {
	// Create a channel to catch signals
	finish := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(finish, os.Interrupt)

	// Make config
	config := infrabin.DefaultConfig()

	flag.BoolVar(
		&config.EnableProxyEndpoint,
		"enable-proxy-endpoint",
		false,
		"If true, enables proxy and aws endpoints",
	)
	flag.Parse()

	// run grpc server in background
	grpcServer := infrabin.NewGRPCServer(config)
	go grpcServer.ListenAndServe()

	// run service server in background
	server := infrabin.NewHTTPServer(
		"server",
		"0.0.0.0:8888",
		infrabin.RegisterInfrabin("/", grpcServer.InfrabinService),
	)
	go server.ListenAndServe()

	// run admin server in background
	admin := infrabin.NewHTTPServer(
		"admin",
		"0.0.0.0:8899",
		infrabin.RegisterHealth("/healthcheck/liveness/", grpcServer.HealthService),
		infrabin.RegisterHealth("/healthcheck/readiness/", grpcServer.HealthService),
	)
	go admin.ListenAndServe()

	// run Prometheus server
	promServer := infrabin.NewHTTPServer(
		"prom",
		"0.0.0.0:8877",
		infrabin.RegisterHandler("/", promhttp.Handler()),
	)
	go promServer.ListenAndServe()

	// wait for SIGINT
	<-finish

	admin.Shutdown()
	server.Shutdown()
	grpcServer.Shutdown()
	promServer.Shutdown()
}
