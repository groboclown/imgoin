// SPDX:Apache-2.0
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	imgoin "github.com/groboclown/imgoin/pkg"
	"go.podman.io/image/v5/transports/alltransports"
)

type ParsedArgs struct {
	sources []*imageOptions
	target  *imageOptions
}

// Parse the arguments.
// Reports to stderr and stdout as issues or help demands.
func ParseArgs(name, version string, args []string) (*ParsedArgs, int) {
	sources := make([]*imageOptions, 0)
	var target *imageOptions = nil

	index := 0
	showHelp := false
	ret := 0
	if len(args) == 0 {
		showHelp = true
		ret = 1
	}

	for index < len(args) {
		arg := args[index]
		switch arg {
		case "-h":
			fallthrough
		case "--help":
			showHelp = true
			ret = -1
			index += 1
		case "-v":
			fallthrough
		case "-V":
			fallthrough
		case "--version":
			printVersion(version)
			return nil, ret
		case "--source":
			source, idx, err := parseImageArgs(version, args, index+1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				showHelp = true
				ret = 1
			} else if source != nil {
				sources = append(sources, source)
			}
			index = idx
		case "--target":
			tgt, idx, err := parseImageArgs(version, args, index+1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				showHelp = true
				ret = 1
			} else if tgt != nil {
				if target != nil {
					fmt.Fprintf(os.Stderr, "Error: can have at most one target image.\n")
					ret = 1
				} else {
					target = tgt
				}
			}
			index = idx
		default:
			fmt.Fprintf(os.Stderr, "Unknown argument (%s)\n", arg)
			showHelp = true
			index += 1
		}
	}

	if showHelp {
		printHelp(name)
	} else {
		if len(sources) == 0 {
			fmt.Fprint(os.Stderr, "Error: must supply at least one (1) source image.\n")
			ret = 2
		}
		if target == nil {
			fmt.Fprint(os.Stderr, "Error: must supply the target image.\n")
			ret = 2
		}
	}
	if ret != 0 {
		return nil, max(0, ret)
	}

	return &ParsedArgs{sources: sources, target: target}, 0
}

var rootlessTransports = []string{
	"containers-storage",
}

// RequiresRootless returns 'true' if any image requires running in rootless mode.
//
// This just hard-codes the transports that need it.
func (p *ParsedArgs) RequiresRootless() bool {
	for _, img := range p.sources {
		if TransportRequireRootless(img.image) {
			return true
		}
	}
	return TransportRequireRootless(p.target.image)
}

func TransportRequireRootless(name string) bool {
	transport := alltransports.TransportFromImageName(name)
	return transport != nil && slices.Contains(rootlessTransports, transport.Name())
}

// AsAction turns the parsed arguments into the values used by the command action.
func (p *ParsedArgs) AsAction(ctx context.Context) (*Action, error) {
	errs := make([]error, 0)
	source_imgs := make([]imgoin.SourceReference, 0)
	for _, src := range p.sources {
		s, err := src.asSource(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %s", src.image, err))
		} else {
			source_imgs = append(source_imgs, s)
		}
	}
	tgt, err := p.target.asTarget(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("%s: %s", p.target.image, err))
		for _, s := range source_imgs {
			var _ = s.Close()
		}
		source_imgs = make([]imgoin.SourceReference, 0)
	}
	return &Action{Sources: source_imgs, Target: tgt}, errors.Join(errs...)
}
