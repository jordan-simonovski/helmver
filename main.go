package main

import (
	"os"

	"github.com/jsimonovski/helmver/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
