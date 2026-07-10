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
			fallthrough
		case "--version":
			printVersion(version)
			return ret
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
		return max(0, ret)
	}

	ctx := context.Background()

	source_imgs := make([]imgoin.SourceReference, len(sources))
	for i, src := range sources {
		s, err := src.asSource(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src.image, err)
			return 3
		}
		defer s.Close()
		source_imgs[i] = s
	}
	tgt, err := target.asTarget(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", target.image, err)
		return 3
	}
	defer tgt.Close()

	// Perform the operation.
	for i, src := range source_imgs {
		src_name := sources[i].image
		fmt.Fprintf(os.Stderr, "Copying %s ...\n", src_name)
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
	fmt.Fprintf(os.Stderr, "Closing out %s ...\n", target.image)
	return 0
}

func printHelp(name string) {
	fmt.Printf(`Usage: %s [-h] --source SETTINGS [--source SETTINGS ...] --target SETTINGS
Where:
   -h, --help       This screen.
   --source         Settings for the source image.  You may specify
                    multiple source images.
   --target			Settings for the target image.

Each '--source' and '--target' argument tells the command to start reading
information about that corresponding source or target.

The SETTINGS takes the form of 'KEY=VALUE'; see the below table for the list
of supported keys and their recognized values.  Some keys allow for setting
multiple values.

Alternatively, you can pass '@FILENAME' to have the command read the settings
from the file named FILENAME.  This file contains 'KEY=VALUE' items, separated
by spaces or newlines.  You can put the value in quotes if it contains a
space, or use \\ to escape a character (like a quote).

General image settings:
  image=IMAGE-LOCATION
      Location of the image, in the format 'transport:name' (see the list of
	  supported transports below).
	  Required for all images.

  auth-file=FILENAME
      Path to a '*/containers/auth.json' file.

  credentials=USERNAME[:PASSWORD]
      Credentials for accessing the registry.

  username=USERNAME
      Username for accessing the registry.

  password=PASSWORD
      Password for accessing the registry.

  registry-token=TOKEN
      Bearer token for accessing the registry.

  certificate-dir=DIRNAME
      Directory containing *.{crt,cert,key} files for contacting the registry.

  tls-verify=yes|no
      Set to 'yes' to require HTTPS + certificate verification.

  anonymous=yes|no
      Set to 'yes' to force anonymous registry access.

  blobs-dir=DIRNAME
      OCI shared blobs directory.

  daemon-host=HOSTNAME[:PORT]
      docker-daemon host for the connection.

Source image settings:

  include-contents=yes|no
      Set to 'yes' to have the target image save all the contents of this
	  source image.  Without it, the target image will store just a reference.
	  Most multi-architecture images store just a reference.

Explicit source image settings:

  You can pass the image setting in the form 'image=sha256:ABC...', which
  allows you to explicitly declare the image reference information.

  manifest-size=SIZE_IN_BYTES
      Number of bytes of the referenced image's manifest.
	
  architecture=ARCH
      Name of the image's target architecture.
  
  os=OS
      Name of the image's target operating system.

  os-version=VERSION
      Version of the image's target operating system.

  os-feature=FEATURE
      A feature required for the image's target operating system.
      You may specify this more than once.

  os-variant=VARIANT
      The image's target operating system variant.

  annotation=KEY:VALUE
      Add a annotation to the image reference.
	  You may provide multiple of these.

Target image settings:

	tag=TAG
	  Give the constructed image a tag.  Only some output transports support this.
	  You may provide multiple of these.

Supported image transports:
`, name)
	for _, name := range transports.ListNames() {
		fmt.Printf("     %s\n", name)
	}
}

func printVersion(version string) {
	fmt.Printf("imgoin v%s\n", version)
}
