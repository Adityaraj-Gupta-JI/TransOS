package main

import (
	"os"

	"github.com/transos/transos/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
