package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/lehigh-university-libraries/cataloger/cmd"
)

const version = "0.1.0"

func main() {
	root := cmd.NewRootCmd()

	// Use fang for beautiful CLI with automatic completions, manpages, --version, etc.
	if err := fang.Execute(
		context.Background(),
		root,
		fang.WithVersion(version),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
