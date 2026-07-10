// SPDX:Apache-2.0
package imgoin_test

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"

	imgoin "github.com/groboclown/imgoin/pkg"
	"github.com/groboclown/imgoin/pkg/fixtures"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

func loadTestSource(t *testing.T, copyContents bool) (*imgoin.SrcImage, []manifest.ListUpdate, []types.BlobInfo) {
	t.Helper()
	tmpdir := t.TempDir()
	tarFile := path.Join(tmpdir, "oci.tar")
	if err := os.WriteFile(tarFile, fixtures.OCI_LINUX_AMD64, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri: fmt.Sprintf("oci-archive:%s", tarFile),
		System:   &types.SystemContext{},
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

	tags := []string{
		"localhost/example/image:alpha",
		"localhost/example/image:beta",
	}
	dest := NewMockImageDestination([]string{imgspecv1.MediaTypeImageIndex})
	tgt := imgoin.NewStdTgtImage(t.Context(), dest, imgoin.DestinationMimesSupportsList(dest.SupportedManifestMIMETypes()))
	tgt.SetTags(tags)
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

	if got := len(dest.PutManifestCalls); got != 1 {
		t.Fatalf("wrong manifest call count before close: %d, expected 1", got)
	}
	if got := len(dest.PutBlobs); got != 2 {
		t.Fatalf("wrong blob call count before close: %d, expected 2", got)
	}
	if !dest.PutBlobFlags[0] {
		t.Fatal("expected config blob to receive isConfig=true")
	}
	if dest.PutBlobFlags[1] {
		t.Fatal("expected layer blob to receive isConfig=false")
	}

	if err := tgt.Close(); err != nil {
		t.Fatal(err)
	}

	if got := len(dest.PutManifestCalls); got != 2 {
		t.Fatalf("wrong manifest call count: %d, expected 2", got)
	}
	if got := len(dest.RepoTags); got != len(tags) {
		t.Fatalf("wrong repo tag count: %d, expected %d", got, len(tags))
	}
	for i, tag := range tags {
		if dest.RepoTags[i] != tag {
			t.Fatalf("wrong repo tag %d: %s, expected %s", i, dest.RepoTags[i], tag)
		}
	}
	if dest.PutManifestCalls[0].InstanceDigest == nil {
		t.Fatal("expected first manifest call to store an instance manifest")
	}
	if dest.PutManifestCalls[1].InstanceDigest != nil {
		t.Fatal("expected second manifest call to store the top-level list")
	}
	if got := len(dest.PutBlobs); got != 2 {
		t.Fatalf("wrong blob call count: %d, expected 2", got)
	}
	if dest.CommitCount != 1 {
		t.Fatalf("wrong commit count: %d, expected 1", dest.CommitCount)
	}
	if dest.CloseCount != 1 {
		t.Fatalf("wrong close count: %d, expected 1", dest.CloseCount)
	}
	if mime := manifest.GuessMIMEType(dest.CommitManifest); mime != imgspecv1.MediaTypeImageIndex {
		t.Fatalf("wrong committed MIME type %s, expected %s", mime, imgspecv1.MediaTypeImageIndex)
	}
	if !bytes.Equal(dest.CommitManifest, dest.PutManifestCalls[1].Data) {
		t.Fatal("committed top-level manifest differs from the manifest written to the destination")
	}
}

func TestStdTgtImageBuffersSingleManifest(t *testing.T) {
	src, manifests, blobs := loadTestSource(t, false)
	defer src.Close()

	tags := []string{"localhost/example/image:single"}
	dest := NewMockImageDestination([]string{manifest.DockerV2Schema2MediaType})
	tgt := imgoin.NewStdTgtImage(t.Context(), dest, imgoin.DestinationMimesSupportsList(dest.SupportedManifestMIMETypes()))
	tgt.SetTags(tags)
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
	if len(dest.PutManifestCalls) != 0 {
		t.Fatalf("unexpected manifest calls before close: %d", len(dest.PutManifestCalls))
	}
	if len(dest.PutBlobs) != 0 {
		t.Fatalf("unexpected blob calls before close: %d", len(dest.PutBlobs))
	}
	// TODO should include some mechanism to test this easier.
	//if len(tgt.manifestBlobs) != 1 {
	//	t.Fatalf("wrong buffered manifest count: %d, expected 1", len(tgt.manifestBlobs))
	//}

	if err := tgt.Close(); err != nil {
		t.Fatal(err)
	}

	if got := len(dest.PutManifestCalls); got != 1 {
		t.Fatalf("wrong manifest call count: %d, expected 1", got)
	}
	if got := len(dest.RepoTags); got != len(tags) {
		t.Fatalf("wrong repo tag count: %d, expected %d", got, len(tags))
	}
	if dest.RepoTags[0] != tags[0] {
		t.Fatalf("wrong repo tag: %s, expected %s", dest.RepoTags[0], tags[0])
	}
	if dest.PutManifestCalls[0].InstanceDigest != nil {
		t.Fatal("expected the final manifest call to write the top-level manifest")
	}
	if got := len(dest.PutBlobs); got != 0 {
		t.Fatalf("unexpected blob call count: %d, expected 0", got)
	}
	if dest.CommitCount != 1 {
		t.Fatalf("wrong commit count: %d, expected 1", dest.CommitCount)
	}
	if dest.CloseCount != 1 {
		t.Fatalf("wrong close count: %d, expected 1", dest.CloseCount)
	}
	if mime := manifest.GuessMIMEType(dest.CommitManifest); mime != imgspecv1.MediaTypeImageManifest {
		t.Fatalf("wrong committed MIME type %s, expected %s", mime, imgspecv1.MediaTypeImageManifest)
	}
}
