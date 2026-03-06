package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version, Commit, and BuildDate are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Gantry version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gantry %s (commit %s, built %s)\n", Version, Commit, BuildDate)
		},
	}
}
