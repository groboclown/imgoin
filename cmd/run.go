// SPDX:Apache-2.0
package cmd

import (
	"context"
	"fmt"
	"os"

	imgoin "github.com/groboclown/imgoin/pkg"
	"go.podman.io/image/v5/transports"
)

func Exec(name, version string, args []string) int {
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
		case "-V":
		case "--version":
			printVersion(name, version)
			return ret
		case "--source":
			source, idx, err := parseImageOptions(args, index+1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				showHelp = true
				ret = 1
			} else if source != nil {
				sources = append(sources, source)
			}
			index = idx
		case "--target":
			tgt, idx, err := parseImageOptions(args, index+1)
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
		return max(0, ret)
	}

	ctx := context.Background()

	source_imgs := make([]imgoin.SourceReference, len(sources))
	for i, src := range sources {
		s, err := src.asSource(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s.\n", err)
			return 3
		}
		defer s.Close()
		source_imgs[i] = s
	}
	tgt, err := target.asTarget(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s.\n", err)
		return 3
	}
	defer tgt.Close()

	// Perform the operation.
	for i, src := range source_imgs {
		src_name := sources[i].image
		fmt.Fprintf(os.Stderr, "Copying %s...\n", src_name)
		mans, err := src.GetManifests(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src_name, err)
			return 4
		}
		for _, man := range mans {
			tgt.AddManifest(man)
		}
		blobs, err := src.GetBlobsToCopy(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src_name, err)
			return 4
		}
		for _, blob := range blobs {
			r, err := src.GetBlob(ctx, blob)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src_name, err)
				return 4
			}
			err = tgt.CopyBlob(blob, r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src_name, err)
				return 4
			}
		}
	}
	return 0
}

func printHelp(name string) {
	fmt.Printf("Usage: %s [-h] --source [KEY=VALUE [KEY=VALUE ...]] --target [KEY=VALUE [KEY=VALUE ...]]\n", name)
	fmt.Println("Where:")
	fmt.Println("   -h, --help       This screen")
	fmt.Println("   --source         Settings for the source image.  You may specify multiple source images.")
	fmt.Println("   --target         Settings for the target image.")
	fmt.Println("   KEY=VALUE        An image setting.  Alternatively, you can provide '@FILENAME' to load the image settings from the given file.")
	fmt.Println("Supported image settings:")
	fmt.Println("   image=IMAGE-LOCATION     Location of the image, in the format 'transport:name'.")
	fmt.Println("Supported image transports:")
	for _, name := range transports.ListNames() {
		fmt.Printf("     %s\n", name)
	}
}

func printVersion(name, version string) {
	fmt.Printf("%s v%s\n", name, version)
}
