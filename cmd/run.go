// SPDX:Apache-2.0
package cmd

import (
	"context"
	"fmt"
	"os"
)

func Exec(ctx context.Context, args *ParsedArgs) int {
	action, err := args.AsAction(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}
	defer action.Close()

	// Perform the operation.
	for _, src := range action.Sources {
		fmt.Fprintf(os.Stderr, "Copying %s ...\n", src.GetName())
		mans, err := src.GetManifests(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src.GetName(), err)
			return 4
		}
		for _, man := range mans {
			action.Target.AddManifest(man)
		}
		blobs, err := src.GetBlobsToCopy(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src.GetName(), err)
			return 4
		}
		for _, blob := range blobs {
			r, err := src.GetBlob(ctx, blob)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src.GetName(), err)
				return 4
			}
			err = action.Target.CopyBlob(blob, r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error with %s: %s.\n", src.GetName(), err)
				return 4
			}
		}
	}
	fmt.Fprintf(os.Stderr, "Closing out %s ...\n", action.Target.GetName())
	return 0
}
