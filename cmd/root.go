package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/maruina/go-infrabin/gen/infrabin/v1/infrabinv1connect"
	"github.com/maruina/go-infrabin/internal/server"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	addr              string
	writeTimeout      time.Duration
	readHeaderTimeout time.Duration
	idleTimeout       time.Duration
	readTimeout       time.Duration
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
	rootCmd.Flags().StringVar(&addr, "addr", ":8888", "TCP address for the server to listen on")
	rootCmd.Flags().DurationVar(&writeTimeout, "write-timeout", 15*time.Second, "HTTP write timeout")
	rootCmd.Flags().DurationVar(&readTimeout, "read-timeout", 60*time.Second, "HTTP read timeout")
	rootCmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 15*time.Second, "HTTP idle timeout")
	rootCmd.Flags().DurationVar(&readHeaderTimeout, "read-header-timeout", 15*time.Second, "HTTP read header timeout")
}

func run(cmd *cobra.Command, args []string) {
	infrabinServer := &server.InfrabinServer{}
	mux := http.NewServeMux()
	path, handler := infrabinv1connect.NewInfrabinServiceHandler(infrabinServer)
	mux.Handle(path, handler)
	httpServer := &http.Server{
		Addr: addr,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		WriteTimeout:      writeTimeout,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server graceful shutdown failed: %v", err)
	} else {
		log.Printf("HTTP server stopped")
	}
}
