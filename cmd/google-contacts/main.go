// Package main is the entry point for google-contacts.
package main

import (
	"fmt"
	"os"

	"google-contacts/internal/cli"
)

func main() {
	cli.Init()

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
