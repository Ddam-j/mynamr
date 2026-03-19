package main

import (
	"os"

	"github.com/Ddam-j/mynamr/internal/cli"
)

func main() {
	app := cli.NewApp(os.Stdin, os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
