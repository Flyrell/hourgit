package main

import (
	"os"

	"github.com/Flyrell/hourgit/internal/cli"
	"github.com/Flyrell/hourgit/internal/project"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	project.SetVersion(version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
