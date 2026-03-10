package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "Display the version of gh-cost-center.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If version is "dev", try to read from VERSION file
		v := version
		if v == "dev" {
			data, err := os.ReadFile("VERSION")
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("reading VERSION file: %w", err)
			}
			if err == nil {
				if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
					v = trimmed
				}
			}
		}
		fmt.Printf("gh-cost-center version %s\n", v)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
