package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/maruina/go-infrabin/pkg/infrabin"
)

func main() {
	// Create a channel to catch signals
	finish := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(finish, os.Interrupt)

	// Make config
	config := &infrabin.Config{}

	flag.BoolVar(
		&config.EnableProxyEndpoint,
		"enable-proxy-endpoint",
		false,
		"If true, enables proxy and aws endpoints",
	)
	flag.Parse()

	// run service server in background
	server := infrabin.NewHTTPServer(config)
	go server.ListenAndServe()

	// run admin server in background
	admin := infrabin.NewAdminServer(config)
	go admin.ListenAndServe()

	// run grpc server in background
	grpcServer := infrabin.NewGRPCServer(config)
	go grpcServer.ListenAndServe()

	// wait for SIGINT
	<-finish

	admin.Shutdown()
	server.Shutdown()
	grpcServer.Shutdown()
}
