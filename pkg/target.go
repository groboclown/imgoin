// SPDX:Apache-2.0

package imgoin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

var _ TargetImage = (*StdTgtImage)(nil)

func (b *BearingImage) AsTargetImage(ctx context.Context) (*StdTgtImage, error) {
	img, err := b.ref.NewImageDestination(ctx, b.sys)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &StdTgtImage{
		ctx:           ctx,
		img:           img,
		supportsList:  DestinationMimesSupportsList(img.SupportedManifestMIMETypes()),
		manifestBlobs: map[digest.Digest][]byte{},
	}, nil
}

func NewStdTgtImage(ctx context.Context, img types.ImageDestination, supportsList bool) *StdTgtImage {
	return &StdTgtImage{
		ctx:           ctx,
		img:           img,
		supportsList:  supportsList,
		manifests:     make([]manifest.ListUpdate, 0),
		manifestBlobs: map[digest.Digest][]byte{},
	}
}

type StdTgtImage struct {
	ctx          context.Context
	img          types.ImageDestination
	supportsList bool
	manifests    []manifest.ListUpdate

	// manifest blobs are generally very small, so this structure can keep these in memory.
	manifestBlobs map[digest.Digest][]byte
}

func (s *StdTgtImage) Close() (retErr error) {
	defer func() {
		if closeErr := s.img.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	manifestBlob, mime, err := s.buildManifest()
	if err != nil {
		return err
	}

	if err := s.img.PutManifest(s.ctx, manifestBlob, nil); err != nil {
		return err
	}

	return s.img.Commit(s.ctx, &staticUnparsedImage{
		ref:      s.img.Reference(),
		manifest: manifestBlob,
		mime:     mime,
	})
}

func (s *StdTgtImage) AddManifest(lu manifest.ListUpdate) error {
	s.manifests = append(s.manifests, lu)
	return nil
}

func (s *StdTgtImage) CopyBlob(bi types.BlobInfo, rc io.ReadCloser) error {
	defer rc.Close()

	if isManifestMediaType(bi.MediaType) {
		data, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		if s.supportsList {
			return s.img.PutManifest(s.ctx, data, &bi.Digest)
		}

		if s.manifestBlobs == nil {
			s.manifestBlobs = map[digest.Digest][]byte{}
		}
		s.manifestBlobs[bi.Digest] = bytes.Clone(data)
		return nil
	}

	_, err := s.img.PutBlob(s.ctx, rc, bi, nil, isConfigMediaType(bi.MediaType))
	return err
}

func (s *StdTgtImage) buildManifest() ([]byte, string, error) {
	if len(s.manifests) == 0 {
		return nil, "", fmt.Errorf("no manifests added")
	}

	if !s.supportsList {
		if len(s.manifests) != 1 {
			return nil, "", fmt.Errorf("destination does not support manifest lists, but %d manifests were added", len(s.manifests))
		}
		manifestBlob, ok := s.manifestBlobs[s.manifests[0].Digest]
		if !ok {
			return nil, "", fmt.Errorf("missing manifest blob for %s", s.manifests[0].Digest)
		}
		mime := s.manifests[0].MediaType
		if mime == "" {
			mime = imgspecv1.MediaTypeImageManifest
		}
		return bytes.Clone(manifestBlob), mime, nil
	}

	components := make([]imgspecv1.Descriptor, len(s.manifests))
	for i, lu := range s.manifests {
		components[i] = listUpdateToDescriptor(lu)
	}
	index := manifest.OCI1IndexFromComponents(components, nil)

	mime := chooseListMIMEType(s.img.SupportedManifestMIMETypes())
	if mime == "" {
		return nil, "", fmt.Errorf("destination does not support manifest lists")
	}
	if mime == imgspecv1.MediaTypeImageIndex {
		blob, err := index.Serialize()
		if err != nil {
			return nil, "", err
		}
		return blob, mime, nil
	}

	list, err := index.ConvertToMIMEType(mime)
	if err != nil {
		return nil, "", err
	}
	blob, err := list.Serialize()
	if err != nil {
		return nil, "", err
	}
	return blob, mime, nil
}

func DestinationMimesSupportsList(mimes []string) bool {
	if len(mimes) == 0 {
		return true
	}
	for _, mime := range mimes {
		if manifest.MIMETypeIsMultiImage(mime) {
			return true
		}
	}
	return false
}

func chooseListMIMEType(mimes []string) string {
	if len(mimes) == 0 {
		return imgspecv1.MediaTypeImageIndex
	}
	for _, mime := range mimes {
		if mime == imgspecv1.MediaTypeImageIndex {
			return imgspecv1.MediaTypeImageIndex
		}
	}
	for _, mime := range mimes {
		if mime == manifest.DockerV2ListMediaType {
			return manifest.DockerV2ListMediaType
		}
	}
	return ""
}

func listUpdateToDescriptor(lu manifest.ListUpdate) imgspecv1.Descriptor {
	desc := imgspecv1.Descriptor{
		MediaType:    lu.MediaType,
		Digest:       lu.Digest,
		Size:         lu.Size,
		Annotations:  maps.Clone(lu.ReadOnly.Annotations),
		ArtifactType: lu.ReadOnly.ArtifactType,
	}
	if lu.ReadOnly.Platform != nil {
		platform := *lu.ReadOnly.Platform
		desc.Platform = &platform
	}
	if listUpdateNeedsZstdAnnotation(lu) {
		if desc.Annotations == nil {
			desc.Annotations = map[string]string{}
		}
		desc.Annotations["io.github.containers.compression.zstd"] = "true"
	}
	return desc
}

func listUpdateNeedsZstdAnnotation(lu manifest.ListUpdate) bool {
	for _, name := range lu.ReadOnly.CompressionAlgorithmNames {
		switch name {
		case "zstd", "zstd:chunked":
			return true
		}
	}
	return false
}

func isConfigMediaType(mediaType string) bool {
	switch mediaType {
	case imgspecv1.MediaTypeImageConfig, manifest.DockerV2Schema2ConfigMediaType:
		return true
	default:
		return false
	}
}

type staticUnparsedImage struct {
	ref      types.ImageReference
	manifest []byte
	mime     string
}

func (s *staticUnparsedImage) Reference() types.ImageReference {
	return s.ref
}

func (s *staticUnparsedImage) Manifest(context.Context) ([]byte, string, error) {
	return s.manifest, s.mime, nil
}

func (s *staticUnparsedImage) Signatures(context.Context) ([][]byte, error) {
	return nil, nil
}
