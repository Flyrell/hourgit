package main

import (
	"os"

	"github.com/Flyrell/hourgit/internal/cli"
	"github.com/Flyrell/hourgit/internal/project"
)

var version = "dev"

func main() {
	cli.SetVersionInfo(version)
	project.SetVersion(version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
