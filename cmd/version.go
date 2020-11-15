package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gitCommit string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print go-infrabin version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("go-infrabin commit %s", gitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
