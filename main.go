package main

import (
	"os"

	"github.com/neoighodaro/blvckhole/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
