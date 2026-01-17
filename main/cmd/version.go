package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version  = "v1.0.0"
	codename = "XMPlus "
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Current version of XMPlus",
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	})
}

func showVersion() {
    fmt.Printf("%s %s \n",codename, version)
}
