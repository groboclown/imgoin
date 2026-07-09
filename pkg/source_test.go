package imgoin_test

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	imgoin "github.com/groboclown/imgoin/pkg"
	"github.com/groboclown/imgoin/pkg/fixtures"
	"go.podman.io/image/v5/manifest"
)

func Test_IndexLoad(t *testing.T) {
	tmpdir := t.TempDir()
	tarFile := path.Join(tmpdir, "index.tar")
	if err := os.WriteFile(tarFile, fixtures.OCI_INDEXED, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri:   fmt.Sprintf("oci-archive:%s", tarFile),
		Connection: &imgoin.RepositoryConnection{},
	})
	if err != nil {
		t.Fatal(err)
	}
	src, err := img.AsSourceImage(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	man, err := src.GetManifests(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(man) != 16 {
		t.Errorf("wrong number of manifests: %d, expected 16", len(man))
	}
}

func Test_DockerImageLoad(t *testing.T) {
	tmpdir := t.TempDir()
	tarFile := path.Join(tmpdir, "docker.tar")
	if err := os.WriteFile(tarFile, fixtures.DOCKER_LINUX_AMD64, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri:   fmt.Sprintf("docker-archive:%s", tarFile),
		Connection: &imgoin.RepositoryConnection{},
	})
	if err != nil {
		t.Fatal(err)
	}
	src, err := img.AsSourceImage(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	man, err := src.GetManifests(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(man) != 1 {
		t.Errorf("wrong number of manifests: %d, expected 1", len(man))
	}
	if man[0].ReadOnly.Platform.Architecture != "amd64" {
		t.Errorf("wrong architecture; found %s, expected amd64", man[0].ReadOnly.Platform.Architecture)
	}
	if man[0].ReadOnly.Platform.OS != "linux" {
		t.Errorf("wrong architecture; found %s, expected linux", man[0].ReadOnly.Platform.OS)
	}
	blobs, err := src.GetBlobsToCopy(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 {
		t.Fatalf("wrong number of blobs: %d, expected 1", len(blobs))
	}
	if blobs[0].Digest != man[0].Digest {
		t.Fatalf("wrong manifest digest %s, expected %s", blobs[0].Digest, man[0].Digest)
	}
	r, err := src.GetBlob(t.Context(), blobs[0])
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		t.Fatal(err)
	}
	digest, err := manifest.Digest(data)
	if err != nil {
		t.Fatal(err)
	}
	if digest != blobs[0].Digest {
		t.Fatalf("wrong blob digest %s, expected %s", digest, blobs[0].Digest)
	}
}

func Test_OCIImageLoad(t *testing.T) {
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
	src, err := img.AsSourceImage(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	man, err := src.GetManifests(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(man) != 1 {
		t.Errorf("wrong number of manifests: %d, expected 1", len(man))
	}
	if man[0].ReadOnly.Platform.Architecture != "amd64" {
		t.Errorf("wrong architecture; found %s, expected amd64", man[0].ReadOnly.Platform.Architecture)
	}
	if man[0].ReadOnly.Platform.OS != "linux" {
		t.Errorf("wrong architecture; found %s, expected linux", man[0].ReadOnly.Platform.OS)
	}
}

func Test_OCIImageCopyContents(t *testing.T) {
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
	src, err := img.AsSourceImage(t.Context(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	blobs, err := src.GetBlobsToCopy(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 3 {
		t.Fatalf("wrong number of blobs: %d, expected 3", len(blobs))
	}
	for i, blob := range blobs {
		r, err := src.GetBlob(t.Context(), blob)
		if err != nil {
			t.Fatalf("blob %d: %v", i, err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatalf("blob %d: %v", i, err)
		}
		digest, err := manifest.Digest(data)
		if err != nil {
			t.Fatalf("blob %d: %v", i, err)
		}
		if digest != blob.Digest {
			t.Fatalf("blob %d: wrong digest %s, expected %s", i, digest, blob.Digest)
		}
	}
}
