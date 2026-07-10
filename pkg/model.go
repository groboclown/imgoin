// SPDX:Apache-2.0
package imgoin

import (
	"context"
	"fmt"
	"io"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"
)

// ImageConnection points to a defined image somewhere, using a supported podman image transport.
type ImageConnection struct {
	ImageUri         string
	Tags             []string
	System           *types.SystemContext
	Policy           *signature.Policy
	RequireSignature bool
}

// SourceReference allows extracting information to put into the target.
type SourceReference interface {
	GetName() string
	GetManifests(ctx context.Context) ([]manifest.ListUpdate, error)
	GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error)
	GetBlob(ctx context.Context, digest types.BlobInfo) (io.ReadCloser, error)
	Close() error
}

// TargetImage allows for storing manifests and data blobs from the multiple source images.
type TargetImage interface {
	GetName() string
	AddManifest(manifest.ListUpdate) error
	CopyBlob(types.BlobInfo, io.ReadCloser) error
	Close() error
}

// SourceManifestReference allows for specifying the meta-data information explicitly.
// This points to the *manifest* information, not that manifest's blob.
// The alternative references the image using the podman images API.
// This can only reference a source manifest, as the target must generate a real thing.
type SourceManifestReference struct {
	name        string
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
		name:   imageName,
		Digest: digest,
		Platform: &imgspecv1.Platform{
			OSFeatures: make([]string, 0),
		},
	}
}

// BearingImage references an image that contains either the manifest index or the data blob.
type BearingImage struct {
	name         string
	sys          *types.SystemContext
	ref          types.ImageReference
	policy       *signature.Policy
	reqSignature bool
	tags         []string
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
		name:         conn.ImageUri,
		tags:         conn.Tags,
		sys:          conn.System,
		policy:       conn.Policy,
		reqSignature: conn.RequireSignature,
		ref:          ref,
	}, nil
}
