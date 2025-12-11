package cli

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Printf("sercha version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
