package main

import (
	"os"

	"github.com/ariel-frischer/autospec/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
