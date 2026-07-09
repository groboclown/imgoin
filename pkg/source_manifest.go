// SPDX:Apache-2.0
package imgoin

import (
	"context"
	"fmt"
	"io"

	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

var _ SourceReference = (*SourceManifestReference)(nil)

func (r *SourceManifestReference) GetManifests(ctx context.Context) ([]manifest.ListUpdate, error) {
	// Allow the user to create the manifest directly.
	item := manifest.ListUpdate{
		Digest:    r.Digest,
		Size:      int64(r.Size),
		MediaType: imgspecv1.MediaTypeImageManifest,
		ReadOnly: struct {
			Platform                  *imgspecv1.Platform
			Annotations               map[string]string
			CompressionAlgorithmNames []string
			ArtifactType              string
		}{
			Platform: &imgspecv1.Platform{
				Architecture: r.Platform.Architecture,
				OS:           r.Platform.OS,
				OSVersion:    r.Platform.OSVersion,
				OSFeatures:   r.Platform.OSFeatures,
				Variant:      r.Platform.Variant,
			},
			Annotations:               r.Annotations,
			CompressionAlgorithmNames: nil,
			ArtifactType:              "",
		},
	}
	return []manifest.ListUpdate{item}, nil
}

func (r *SourceManifestReference) GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error) {
	return nil, nil
}

func (r *SourceManifestReference) GetBlob(ctx context.Context, blob types.BlobInfo) (io.ReadCloser, error) {
	return nil, fmt.Errorf("no such blob")
}

func (r *SourceManifestReference) Close() error {
	return nil
}
