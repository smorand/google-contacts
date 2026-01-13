// Package cli provides the command-line interface for google-contacts.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
const Version = "0.1.0"

// RootCmd is the root command for the CLI.
var RootCmd = &cobra.Command{
	Use:   "google-contacts",
	Short: "Google Contacts Manager - Manage Google Contacts",
	Long:  "Create, search, and manage Google Contacts using Google People API v1",
}

// Command definitions
var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("google-contacts version %s\n", Version)
		},
	}
)

// Init initializes the CLI commands and flags.
func Init() {
	// Add version flag to root command
	RootCmd.Version = Version
	RootCmd.SetVersionTemplate("google-contacts version {{.Version}}\n")

	// Register commands
	RootCmd.AddCommand(versionCmd)
}
