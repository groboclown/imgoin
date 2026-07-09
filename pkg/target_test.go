// SPDX:Apache-2.0
package imgoin_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	imgoin "github.com/groboclown/imgoin/pkg"
	"github.com/groboclown/imgoin/pkg/fixtures"
	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

// Manifest spy
type recordedManifestCall struct {
	data           []byte
	instanceDigest *digest.Digest
}

// ImageDestination spy
type recordingDestination struct {
	supported        []string
	putBlobs         []types.BlobInfo
	putBlobFlags     []bool
	putManifestCalls []recordedManifestCall
	commitManifest   []byte
	commitMime       string
	commitCount      int
	closeCount       int
}

var _ types.ImageDestination = (*recordingDestination)(nil)

func (d *recordingDestination) Reference() types.ImageReference { return nil }
func (d *recordingDestination) Close() error {
	d.closeCount++
	return nil
}
func (d *recordingDestination) SupportedManifestMIMETypes() []string     { return d.supported }
func (d *recordingDestination) SupportsSignatures(context.Context) error { return nil }
func (d *recordingDestination) DesiredLayerCompression() types.LayerCompression {
	return types.PreserveOriginal
}
func (d *recordingDestination) AcceptsForeignLayerURLs() bool        { return true }
func (d *recordingDestination) MustMatchRuntimeOS() bool             { return false }
func (d *recordingDestination) IgnoresEmbeddedDockerReference() bool { return false }
func (d *recordingDestination) HasThreadSafePutBlob() bool           { return true }
func (d *recordingDestination) TryReusingBlob(context.Context, types.BlobInfo, types.BlobInfoCache, bool) (bool, types.BlobInfo, error) {
	return false, types.BlobInfo{}, nil
}
func (d *recordingDestination) PutBlob(_ context.Context, stream io.Reader, inputInfo types.BlobInfo, _ types.BlobInfoCache, isConfig bool) (types.BlobInfo, error) {
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
	d.putBlobs = append(d.putBlobs, types.BlobInfo{
		Digest:    inputInfo.Digest,
		Size:      int64(len(data)),
		MediaType: inputInfo.MediaType,
	})
	d.putBlobFlags = append(d.putBlobFlags, isConfig)
	return types.BlobInfo{
		Digest:    inputInfo.Digest,
		Size:      int64(len(data)),
		MediaType: inputInfo.MediaType,
	}, nil
}
func (d *recordingDestination) PutManifest(_ context.Context, data []byte, instanceDigest *digest.Digest) error {
	if instanceDigest != nil {
		matches, err := manifest.MatchesDigest(data, *instanceDigest)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("manifest digest mismatch for %s", instanceDigest.String())
		}
	}
	call := recordedManifestCall{data: bytes.Clone(data)}
	if instanceDigest != nil {
		digestValue := *instanceDigest
		call.instanceDigest = &digestValue
	}
	d.putManifestCalls = append(d.putManifestCalls, call)
	return nil
}
func (d *recordingDestination) PutSignatures(context.Context, [][]byte, *digest.Digest) error {
	return nil
}
func (d *recordingDestination) Commit(ctx context.Context, unparsed types.UnparsedImage) error {
	d.commitCount++
	if unparsed == nil {
		return fmt.Errorf("commit requires an unparsed image")
	}
	data, mime, err := unparsed.Manifest(ctx)
	if err != nil {
		return err
	}
	d.commitManifest = bytes.Clone(data)
	d.commitMime = mime
	return nil
}

func loadTestSource(t *testing.T, copyContents bool) (*imgoin.SrcImage, []manifest.ListUpdate, []types.BlobInfo) {
	t.Helper()
	tmpdir := t.TempDir()
	tarFile := path.Join(tmpdir, "oci.tar")
	if err := os.WriteFile(tarFile, fixtures.OCI_LINUX_AMD64, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri:   fmt.Sprintf("oci-archive:%s", tarFile),
		Connection: &imgoin.RepositoryConnection{},
	})
	if err != nil {
		t.Fatal(err)
	}
	src, err := img.AsSourceImage(t.Context(), copyContents)
	if err != nil {
		t.Fatal(err)
	}
	manifests, err := src.GetManifests(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := src.GetBlobsToCopy(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return src, manifests, blobs
}

func TestStdTgtImageCopiesListAndBlobs(t *testing.T) {
	src, manifests, blobs := loadTestSource(t, true)
	defer src.Close()

	dest := &recordingDestination{supported: []string{imgspecv1.MediaTypeImageIndex}}
	tgt := imgoin.NewStdTgtImage(t.Context(), dest, imgoin.DestinationMimesSupportsList(dest.SupportedManifestMIMETypes()))
	for _, manifestUpdate := range manifests {
		if err := tgt.AddManifest(manifestUpdate); err != nil {
			t.Fatal(err)
		}
	}
	for i, blob := range blobs {
		rc, err := src.GetBlob(t.Context(), blob)
		if err != nil {
			t.Fatalf("blob %d: %v", i, err)
		}
		if err := tgt.CopyBlob(blob, rc); err != nil {
			t.Fatalf("blob %d: %v", i, err)
		}
	}

	if got := len(dest.putManifestCalls); got != 1 {
		t.Fatalf("wrong manifest call count before close: %d, expected 1", got)
	}
	if got := len(dest.putBlobs); got != 2 {
		t.Fatalf("wrong blob call count before close: %d, expected 2", got)
	}
	if !dest.putBlobFlags[0] {
		t.Fatal("expected config blob to receive isConfig=true")
	}
	if dest.putBlobFlags[1] {
		t.Fatal("expected layer blob to receive isConfig=false")
	}

	if err := tgt.Close(); err != nil {
		t.Fatal(err)
	}

	if got := len(dest.putManifestCalls); got != 2 {
		t.Fatalf("wrong manifest call count: %d, expected 2", got)
	}
	if dest.putManifestCalls[0].instanceDigest == nil {
		t.Fatal("expected first manifest call to store an instance manifest")
	}
	if dest.putManifestCalls[1].instanceDigest != nil {
		t.Fatal("expected second manifest call to store the top-level list")
	}
	if got := len(dest.putBlobs); got != 2 {
		t.Fatalf("wrong blob call count: %d, expected 2", got)
	}
	if dest.commitCount != 1 {
		t.Fatalf("wrong commit count: %d, expected 1", dest.commitCount)
	}
	if dest.closeCount != 1 {
		t.Fatalf("wrong close count: %d, expected 1", dest.closeCount)
	}
	if mime := manifest.GuessMIMEType(dest.commitManifest); mime != imgspecv1.MediaTypeImageIndex {
		t.Fatalf("wrong committed MIME type %s, expected %s", mime, imgspecv1.MediaTypeImageIndex)
	}
	if !bytes.Equal(dest.commitManifest, dest.putManifestCalls[1].data) {
		t.Fatal("committed top-level manifest differs from the manifest written to the destination")
	}
}

func TestStdTgtImageBuffersSingleManifest(t *testing.T) {
	src, manifests, blobs := loadTestSource(t, false)
	defer src.Close()

	dest := &recordingDestination{supported: []string{manifest.DockerV2Schema2MediaType}}
	tgt := imgoin.NewStdTgtImage(t.Context(), dest, imgoin.DestinationMimesSupportsList(dest.SupportedManifestMIMETypes()))
	for _, manifestUpdate := range manifests {
		if err := tgt.AddManifest(manifestUpdate); err != nil {
			t.Fatal(err)
		}
	}
	if len(blobs) != 1 {
		t.Fatalf("wrong number of blobs: %d, expected 1", len(blobs))
	}
	rc, err := src.GetBlob(t.Context(), blobs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := tgt.CopyBlob(blobs[0], rc); err != nil {
		t.Fatal(err)
	}
	if len(dest.putManifestCalls) != 0 {
		t.Fatalf("unexpected manifest calls before close: %d", len(dest.putManifestCalls))
	}
	if len(dest.putBlobs) != 0 {
		t.Fatalf("unexpected blob calls before close: %d", len(dest.putBlobs))
	}
	// TODO should include some mechanism to test this easier.
	//if len(tgt.manifestBlobs) != 1 {
	//	t.Fatalf("wrong buffered manifest count: %d, expected 1", len(tgt.manifestBlobs))
	//}

	if err := tgt.Close(); err != nil {
		t.Fatal(err)
	}

	if got := len(dest.putManifestCalls); got != 1 {
		t.Fatalf("wrong manifest call count: %d, expected 1", got)
	}
	if dest.putManifestCalls[0].instanceDigest != nil {
		t.Fatal("expected the final manifest call to write the top-level manifest")
	}
	if got := len(dest.putBlobs); got != 0 {
		t.Fatalf("unexpected blob call count: %d, expected 0", got)
	}
	if dest.commitCount != 1 {
		t.Fatalf("wrong commit count: %d, expected 1", dest.commitCount)
	}
	if dest.closeCount != 1 {
		t.Fatalf("wrong close count: %d, expected 1", dest.closeCount)
	}
	if mime := manifest.GuessMIMEType(dest.commitManifest); mime != imgspecv1.MediaTypeImageManifest {
		t.Fatalf("wrong committed MIME type %s, expected %s", mime, imgspecv1.MediaTypeImageManifest)
	}
}
