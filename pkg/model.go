// SPDX:Apache-2.0
package imgoin

import (
	"context"
	"fmt"
	"io"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"
)

// ImageConnection points to a defined image somewhere, using a supported podman image transport.
type ImageConnection struct {
	ImageUri string
	Tags     []string
	System   *types.SystemContext
}

// SourceReference allows extracting information to put into the target.
type SourceReference interface {
	GetManifests(ctx context.Context) ([]manifest.ListUpdate, error)
	GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error)
	GetBlob(ctx context.Context, digest types.BlobInfo) (io.ReadCloser, error)
	Close() error
}

// TargetImage allows for storing manifests and data blobs from the multiple source images.
type TargetImage interface {
	AddManifest(manifest.ListUpdate) error
	CopyBlob(types.BlobInfo, io.ReadCloser) error
	Close() error
}

// SourceManifestReference allows for specifying the meta-data information explicitly.
// This points to the *manifest* information, not that manifest's blob.
// The alternative references the image using the podman images API.
// This can only reference a source manifest, as the target must generate a real thing.
type SourceManifestReference struct {
	Digest      digest.Digest
	Size        uint64
	Annotations map[string]string
	Platform    *imgspecv1.Platform
}

// AsSourceManifestReference constructs a SourceManifestReference based on the image name.
// It does not populate the size, annotations, or platform.
// If the `imageName` does not use a digest prefix, then this returns nil.
func AsSourceManifestReference(imageName string) *SourceManifestReference {
	digest, err := digest.Parse(imageName)
	if err != nil {
		return nil
	}
	return &SourceManifestReference{
		Digest: digest,
		Platform: &imgspecv1.Platform{
			OSFeatures: make([]string, 0),
		},
	}
}

// BearingImage references an image that contains either the manifest index or the data blob.
type BearingImage struct {
	sys  *types.SystemContext
	ref  types.ImageReference
	tags []string
}

// AsBearingImage constructs a BearingImage from an image connection.
func AsBearingImage(conn ImageConnection) (*BearingImage, error) {
	if conn.System == nil {
		return nil, fmt.Errorf("BUG nil connection information")
	}
	ref, err := alltransports.ParseImageName(conn.ImageUri)
	if err != nil {
		return nil, err
	}
	return &BearingImage{
		tags: conn.Tags,
		sys:  conn.System,
		ref:  ref,
	}, nil
}

func ptrAsStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrAsBool(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func ptrAsOptionalBool(b *bool) types.OptionalBool {
	if b == nil {
		return types.OptionalBoolUndefined
	}
	if *b {
		return types.OptionalBoolTrue
	}
	return types.OptionalBoolFalse
}
