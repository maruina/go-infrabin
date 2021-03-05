package main

import (
	"os"

	"github.com/maruina/go-infrabin/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1) //nolint:gomnd // custom error codes wouldn't provide much value
	}
}
