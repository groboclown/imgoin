// SPDX:Apache-2.0

package imgoin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/docker/reference"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

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
		tags:          slices.Clone(b.tags),
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
	tags         []string
	supportsList bool
	manifests    []manifest.ListUpdate

	// manifest blobs are generally very small, so this structure can keep these in memory.
	manifestBlobs map[digest.Digest][]byte
}

var _ TargetImage = (*StdTgtImage)(nil)

func (s *StdTgtImage) Close() (retErr error) {
	// Closing the image means committing it to disk.
	defer func() {
		if closeErr := s.img.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	if err := s.addRepoTags(); err != nil {
		return err
	}

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

func (s *StdTgtImage) SetTags(tags []string) {
	s.tags = slices.Clone(tags)
}

func (s *StdTgtImage) CopyBlob(bi types.BlobInfo, rc io.ReadCloser) error {
	defer rc.Close()

	if isManifestMediaType(bi.MediaType) {
		// Manifest media should have a small size, so this can read it into memory.
		data, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		// If the output image format does not directly support a manifest list,
		// then these need to be added later.  Otherwise, add the manifest
		// now.
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

func (s *StdTgtImage) addRepoTags() error {
	if len(s.tags) == 0 {
		return nil
	}

	// The ability to add repository tags to the output
	// only applies to archive files.  Currently, that
	// means it only the transports 'docker-archive',
	// 'docker', and 'tarfile'.

	type repoTagAdder interface {
		AddRepoTags([]reference.NamedTagged)
	}

	adder, ok := s.img.(repoTagAdder)
	if !ok {
		return nil
	}

	repoTags := make([]reference.NamedTagged, 0, len(s.tags))
	for _, tag := range s.tags {
		named, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return fmt.Errorf("parsing target tag %q: %w", tag, err)
		}
		tagged, ok := reference.TagNameOnly(named).(reference.NamedTagged)
		if !ok {
			return fmt.Errorf("target tag %q does not include a tag", tag)
		}
		repoTags = append(repoTags, tagged)
	}

	adder.AddRepoTags(repoTags)
	return nil
}

// buildManifest creates the index manifest for the image, based on its current state.
func (s *StdTgtImage) buildManifest() ([]byte, string, error) {
	if len(s.manifests) == 0 {
		return nil, "", fmt.Errorf("no manifests added")
	}

	// Image types that don't support lists have a single type manifest.
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

	// Create the index from the components.
	components := make([]imgspecv1.Descriptor, len(s.manifests))
	for i, lu := range s.manifests {
		components[i] = listUpdateToDescriptor(lu)
	}
	index := manifest.OCI1IndexFromComponents(components, nil)

	// Figure out the right way to serialize the index manifest based on
	// its mime type.
	mime := chooseListMIMEType(s.img.SupportedManifestMIMETypes())
	if mime == "" {
		return nil, "", fmt.Errorf("destination does not support manifest lists")
	}
	if mime == imgspecv1.MediaTypeImageIndex {
		// The generated index mime type matches the expected output mime type.
		blob, err := index.Serialize()
		if err != nil {
			return nil, "", err
		}
		return blob, mime, nil
	}

	// Need to convert it.
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

// DestinationMimesSupportsList determines if the list of mimes includes multi-image types.
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

// listMimePriority contains an internal priority for the mime type to use
// when constructing the target list index.
var listMimePriority = []string{
	imgspecv1.MediaTypeImageIndex,
	manifest.DockerV2ListMediaType,
}

// chooseListMIMEType chooses the best image mime type from the parameter list.
func chooseListMIMEType(mimes []string) string {
	if len(mimes) == 0 {
		return imgspecv1.MediaTypeImageIndex
	}
	for _, mime := range listMimePriority {
		if slices.Contains(mimes, mime) {
			return mime
		}
	}
	return ""
}

// listUpdateToDescriptor turns a ListUpdate from a source image into a target Descriptor manifest.
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

// listUpdateNeedsZstdAnnotation determines if this image uses zstd compression.
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

// An unparsed image used for persisting the target image.
type staticUnparsedImage struct {
	ref      types.ImageReference
	manifest []byte
	mime     string
}

var _ types.UnparsedImage = (*staticUnparsedImage)(nil)

func (s *staticUnparsedImage) Reference() types.ImageReference {
	return s.ref
}

func (s *staticUnparsedImage) Manifest(context.Context) ([]byte, string, error) {
	return s.manifest, s.mime, nil
}

func (s *staticUnparsedImage) Signatures(context.Context) ([][]byte, error) {
	return nil, nil
}
