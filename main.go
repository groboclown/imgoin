//SPDX:Apache-2.0

package main

import (
	_ "embed"
	"os"

	"github.com/groboclown/imgoin/cmd"
)

//go:embed version.txt
var Version string

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	return cmd.Exec(args[0], Version, args[1:])
}
