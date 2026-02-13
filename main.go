package main

import (
	"os"

	"github.com/jordan-simonovski/helmver/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
