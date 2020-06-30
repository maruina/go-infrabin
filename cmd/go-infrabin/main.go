package main

import (
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
	
	// run service server in background
	server := infrabin.NewHTTPServer()
	go server.ListenAndServe()

	// run admin server in background
	admin := infrabin.NewAdminServer()
	go admin.ListenAndServe()

	// wait for SIGINT
	<-finish

	admin.Shutdown()
	server.Shutdown()
}
