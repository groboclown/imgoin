// SPDX:Apache-2.0
package imgoin_test

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	"go.podman.io/image/v5/docker/reference"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

// MockImageDestination mocks up an ImageDestination.
type MockImageDestination struct {
	supported        []string
	RepoTags         []string
	PutBlobs         []types.BlobInfo
	PutBlobFlags     []bool
	PutManifestCalls []AddedManifest
	CommitManifest   []byte
	CommitMime       string
	CommitCount      int
	CloseCount       int
}

var _ types.ImageDestination = (*MockImageDestination)(nil)

func NewMockImageDestination(supportedMimes []string) *MockImageDestination {
	return &MockImageDestination{supported: supportedMimes}
}

// AddedManifest keeps track of calls for adding manifests.
type AddedManifest struct {
	Data           []byte
	InstanceDigest *digest.Digest
}

func (d *MockImageDestination) Reference() types.ImageReference { return nil }

func (d *MockImageDestination) Close() error {
	d.CloseCount++
	return nil
}

func (d *MockImageDestination) SupportedManifestMIMETypes() []string { return d.supported }

func (d *MockImageDestination) SupportsSignatures(context.Context) error { return nil }

func (d *MockImageDestination) DesiredLayerCompression() types.LayerCompression {
	return types.PreserveOriginal
}

func (d *MockImageDestination) AddRepoTags(tags []reference.NamedTagged) {
	for _, tag := range tags {
		d.RepoTags = append(d.RepoTags, tag.String())
	}
}

func (d *MockImageDestination) AcceptsForeignLayerURLs() bool { return true }

func (d *MockImageDestination) MustMatchRuntimeOS() bool { return false }

func (d *MockImageDestination) IgnoresEmbeddedDockerReference() bool { return false }

func (d *MockImageDestination) HasThreadSafePutBlob() bool { return true }

func (d *MockImageDestination) TryReusingBlob(context.Context, types.BlobInfo, types.BlobInfoCache, bool) (bool, types.BlobInfo, error) {
	return false, types.BlobInfo{}, nil
}

func (d *MockImageDestination) PutBlob(_ context.Context, stream io.Reader, inputInfo types.BlobInfo, _ types.BlobInfoCache, isConfig bool) (types.BlobInfo, error) {
	data, err := io.ReadAll(stream)
	if err != nil {
		return types.BlobInfo{}, err
	}
	if inputInfo.Digest != "" {
		digestValue, err := manifest.Digest(data)
		if err != nil {
			return types.BlobInfo{}, err
		}
		if digestValue != inputInfo.Digest {
			return types.BlobInfo{}, fmt.Errorf("blob digest mismatch: got %s, expected %s", digestValue, inputInfo.Digest)
		}
	}
	d.PutBlobs = append(d.PutBlobs, types.BlobInfo{
		Digest:    inputInfo.Digest,
		Size:      int64(len(data)),
		MediaType: inputInfo.MediaType,
	})
	d.PutBlobFlags = append(d.PutBlobFlags, isConfig)
	return types.BlobInfo{
		Digest:    inputInfo.Digest,
		Size:      int64(len(data)),
		MediaType: inputInfo.MediaType,
	}, nil
}

func (d *MockImageDestination) PutManifest(_ context.Context, data []byte, instanceDigest *digest.Digest) error {
	if instanceDigest != nil {
		matches, err := manifest.MatchesDigest(data, *instanceDigest)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("manifest digest mismatch for %s", instanceDigest.String())
		}
	}
	call := AddedManifest{Data: bytes.Clone(data)}
	if instanceDigest != nil {
		digestValue := *instanceDigest
		call.InstanceDigest = &digestValue
	}
	d.PutManifestCalls = append(d.PutManifestCalls, call)
	return nil
}

func (d *MockImageDestination) PutSignatures(context.Context, [][]byte, *digest.Digest) error {
	return fmt.Errorf("signatures not supported right now")
}

func (d *MockImageDestination) Commit(ctx context.Context, unparsed types.UnparsedImage) error {
	d.CommitCount++
	if unparsed == nil {
		return fmt.Errorf("commit requires an unparsed image")
	}
	data, mime, err := unparsed.Manifest(ctx)
	if err != nil {
		return err
	}
	d.CommitManifest = bytes.Clone(data)
	d.CommitMime = mime
	return nil
}
