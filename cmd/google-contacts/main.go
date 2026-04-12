// Package main is the entry point for google-contacts.
package main

import (
	"fmt"
	"os"

	"google-contacts/internal/cli"
	"google-contacts/internal/observability"
)

func main() {
	observability.InitLogger(os.Getenv("LOG_LEVEL"))
	cli.Init()

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
