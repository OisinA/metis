package cmd

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:   "metis",
		Short: "Simple container orchestration",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(controllerCmd)
}
