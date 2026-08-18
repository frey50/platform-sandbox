package main

import (
	"fmt"
	"os"

	"github.com/frey50/platform-sandbox/cmd/platformctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
