package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect"
	"github.com/maruina/go-infrabin/internal/server"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	addr              string
	drainTimeout      time.Duration
	idleTimeout       time.Duration
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-infrabin",
	Short: "go-infrabin is an HTTP/gRPC server to test your infrastructure.",
	Run:   run,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().DurationVar(&drainTimeout, "drain-timeout", 60*time.Second, "Drain timeout to wait for in-flight connections to terminate before closing the connection")
	rootCmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 15*time.Second, "HTTP idle timeout")
	rootCmd.Flags().DurationVar(&readHeaderTimeout, "read-header-timeout", 15*time.Second, "HTTP read header timeout")
	rootCmd.Flags().DurationVar(&readTimeout, "read-timeout", 60*time.Second, "HTTP read timeout")
	rootCmd.Flags().DurationVar(&writeTimeout, "write-timeout", 15*time.Second, "HTTP write timeout")
	rootCmd.Flags().StringVar(&addr, "addr", ":8888", "TCP address for the server to listen on")

}

func run(cmd *cobra.Command, args []string) {
	infrabinServer := &server.InfrabinServer{}
	mux := http.NewServeMux()
	path, handler := infrabinv1connect.NewInfrabinServiceHandler(infrabinServer)
	mux.Handle(path, handler)
	srv := &http.Server{
		Addr: addr,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		WriteTimeout:      writeTimeout,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on addr: %v", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Server started")

	<-done
	log.Print("Server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Print("Server shutdown gracefully")

}
