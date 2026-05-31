package cmd

import "github.com/spf13/cobra"

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import content from external sources",
}

func init() {
	rootCmd.AddCommand(importCmd)
}
