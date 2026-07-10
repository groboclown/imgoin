//SPDX:Apache-2.0

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/groboclown/imgoin/cmd"
	"go.podman.io/storage/pkg/reexec"
)

//go:embed version.txt
var Version string

func main() {
	if reexec.Init() {
		// Already performed execution, so no need to run again.
		return
	}
	os.Exit(run(os.Args))
}

func run(args []string) int {
	parsed, res := cmd.ParseArgs(args[0], Version, args[1:])
	if res != 0 {
		os.Exit(res)
	}
	if parsed.RequiresRootless() {
		// Need to restart in rootless mode, if the command isn't already running
		// in that mode.  This allows the program to access Podman images on Linux systems.
		if err := cmd.MaybeReexec(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return 5
		}
	}
	return cmd.Exec(context.Background(), parsed)
}
