package main

import (
	"context"
	"os"

	"github.com/prismgo/installer/internal/cli"
)

func main() {
	// Execute receives the process arguments without argv[0] so Cobra parses only user input.
	if err := cli.Execute(context.Background(), os.Args[1:]); err != nil {
		if _, writeErr := os.Stderr.WriteString(err.Error() + "\n"); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
