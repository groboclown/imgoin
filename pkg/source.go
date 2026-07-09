// SPDX:Apache-2.0
package imgoin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

var _ SourceReference = (*SourceManifestReference)(nil)

func (r *SourceManifestReference) GetManifests(ctx context.Context) ([]manifest.ListUpdate, error) {
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

type SrcImage struct {
	img          types.ImageSource
	copyContents bool
}

func (b *BearingImage) AsSourceImage(ctx context.Context, copyContents bool) (*SrcImage, error) {
	img, err := b.ref.NewImageSource(ctx, b.sys)
	if err != nil {
		return nil, err
	}
	return &SrcImage{img, copyContents}, nil
}

var _ SourceReference = (*SrcImage)(nil)

func (r *SrcImage) GetManifests(ctx context.Context) ([]manifest.ListUpdate, error) {
	raw, mime, err := r.img.GetManifest(ctx, nil)
	if err != nil {
		return nil, err
	}
	mt := sourceManifestMIMEType(raw, mime)
	if manifest.MIMETypeIsMultiImage(mt) {
		list, err := manifest.ListFromBlob(raw, mt)
		if err != nil {
			return nil, err
		}
		ret := make([]manifest.ListUpdate, 0, len(list.Instances()))
		for _, digest := range list.Instances() {
			inst, err := list.Instance(digest)
			if err != nil {
				return nil, err
			}
			// Use the list metadata directly; some archives carry only the index blob.
			instUpdate := manifest.ListUpdate{
				Digest:    inst.Digest,
				Size:      inst.Size,
				MediaType: inst.MediaType,
				ReadOnly:  inst.ReadOnly,
			}
			ret = append(ret, instUpdate)
		}
		return ret, nil
	}
	man, err := manifest.FromBlob(raw, mt)
	if err != nil {
		return nil, err
	}
	manifestDigest, err := manifest.Digest(raw)
	if err != nil {
		return nil, err
	}
	item := manifest.ListUpdate{
		Digest:    manifestDigest,
		Size:      int64(len(raw)),
		MediaType: mt,
	}
	if ociMan, err := manifest.OCI1FromManifest(raw); err == nil {
		if len(ociMan.Annotations) > 0 {
			item.ReadOnly.Annotations = maps.Clone(ociMan.Annotations)
		}
		item.ReadOnly.ArtifactType = ociMan.ArtifactType
	}
	config := man.ConfigInfo()
	if item.ReadOnly.Annotations == nil && len(config.Annotations) > 0 {
		item.ReadOnly.Annotations = maps.Clone(config.Annotations)
	}
	if config.Digest != "" {
		configMan, err := parseConfig(ctx, r.img, config)
		if err != nil {
			return nil, err
		}
		item.ReadOnly.Platform = &imgspecv1.Platform{
			Architecture: configMan.Architecture,
			OS:           configMan.OS,
			OSVersion:    configMan.OSVersion,
			OSFeatures:   configMan.OSFeatures,
			Variant:      configMan.Variant,
		}
	}
	item.ReadOnly.CompressionAlgorithmNames = compressionNamesFromLayerInfos(man.LayerInfos())
	return []manifest.ListUpdate{item}, nil
}

func (r *SrcImage) GetBlob(ctx context.Context, blob types.BlobInfo) (io.ReadCloser, error) {
	if isManifestMediaType(blob.MediaType) {
		if rc, _, err := r.img.GetBlob(ctx, blob, nil); err == nil {
			data, readErr := io.ReadAll(rc)
			rc.Close()
			if readErr == nil {
				if digestValue, digestErr := manifest.Digest(data); digestErr == nil && digestValue == blob.Digest {
					return io.NopCloser(bytes.NewReader(data)), nil
				}
			}
		}

		raw, _, err := r.manifestBlob(ctx, &blob.Digest)
		if err == nil {
			if digestValue, digestErr := manifest.Digest(raw); digestErr == nil && digestValue == blob.Digest {
				return io.NopCloser(bytes.NewReader(raw)), nil
			}
		}

		raw, _, err = r.manifestBlob(ctx, nil)
		if err != nil {
			return nil, err
		}
		digestValue, err := manifest.Digest(raw)
		if err != nil {
			return nil, err
		}
		if digestValue != blob.Digest {
			return nil, fmt.Errorf("manifest digest mismatch: got %s, expected %s", digestValue, blob.Digest)
		}
		return io.NopCloser(bytes.NewReader(raw)), nil
	}
	rc, _, err := r.img.GetBlob(ctx, blob, nil)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (r *SrcImage) GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error) {
	raw, mime, err := r.img.GetManifest(ctx, nil)
	if err != nil {
		return nil, err
	}
	mt := sourceManifestMIMEType(raw, mime)
	if manifest.MIMETypeIsMultiImage(mt) {
		list, err := manifest.ListFromBlob(raw, mt)
		if err != nil {
			return nil, err
		}

		ret := make([]types.BlobInfo, 0, len(list.Instances()))
		for _, digestValue := range list.Instances() {
			instRaw, instMime, err := r.manifestBlob(ctx, &digestValue)
			if err != nil {
				return nil, err
			}
			instDigest, err := manifest.Digest(instRaw)
			if err != nil {
				return nil, err
			}
			ret = append(ret, types.BlobInfo{
				Digest:    instDigest,
				Size:      int64(len(instRaw)),
				MediaType: sourceManifestMIMEType(instRaw, instMime),
			})
			if !r.copyContents {
				continue
			}

			instType := sourceManifestMIMEType(instRaw, instMime)
			man, err := manifest.FromBlob(instRaw, instType)
			if err != nil {
				return nil, err
			}
			config := man.ConfigInfo()
			if config.Digest != "" {
				ret = append(ret, config)
			}
			layers := manifestLayerInfosToBlobInfos(man.LayerInfos())
			copyLayers, err := r.img.LayerInfosForCopy(ctx, &digestValue)
			if err != nil {
				return nil, err
			}
			if copyLayers != nil {
				layers = copyLayers
			}
			ret = append(ret, layers...)
		}
		return ret, nil
	}

	man, err := manifest.FromBlob(raw, mt)
	if err != nil {
		return nil, err
	}
	manifestDigest, err := manifest.Digest(raw)
	if err != nil {
		return nil, err
	}
	ret := []types.BlobInfo{{
		Digest:    manifestDigest,
		Size:      int64(len(raw)),
		MediaType: mt,
	}}
	if !r.copyContents {
		return ret, nil
	}

	config := man.ConfigInfo()
	if config.Digest != "" {
		ret = append(ret, config)
	}
	layers := manifestLayerInfosToBlobInfos(man.LayerInfos())
	copyLayers, err := r.img.LayerInfosForCopy(ctx, nil)
	if err != nil {
		return nil, err
	}
	if copyLayers != nil {
		layers = copyLayers
	}
	ret = append(ret, layers...)
	return ret, nil
}

func (r *SrcImage) Close() error {
	return r.img.Close()
}

func parseConfig(ctx context.Context, img types.ImageSource, config types.BlobInfo) (*SimpleConfig, error) {
	// docker media type config info blob contains the json formatted file.
	configReader, _, err := img.GetBlob(ctx, config, nil)
	if err != nil {
		return nil, err
	}
	defer configReader.Close()
	configBlob, err := io.ReadAll(configReader)
	if err != nil {
		return nil, err
	}
	var sc SimpleConfig
	if err := json.Unmarshal(configBlob, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

func (r *SrcImage) manifestBlob(ctx context.Context, instanceDigest *digest.Digest) ([]byte, string, error) {
	return r.img.GetManifest(ctx, instanceDigest)
}

func sourceManifestMIMEType(raw []byte, mime string) string {
	if guessed := manifest.GuessMIMEType(raw); guessed != "" {
		return guessed
	}
	if mime != "" {
		return mime
	}
	return imgspecv1.MediaTypeImageManifest
}

func isManifestMediaType(mediaType string) bool {
	switch mediaType {
	case imgspecv1.MediaTypeImageManifest,
		imgspecv1.MediaTypeImageIndex,
		manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType,
		manifest.DockerV2Schema2MediaType,
		manifest.DockerV2ListMediaType:
		return true
	default:
		return false
	}
}

func compressionNamesFromLayerInfos(layerInfos []manifest.LayerInfo) []string {
	names := make([]string, 0, 2)
	seen := make(map[string]struct{})
	for _, layerInfo := range layerInfos {
		name := compressionNameForMediaType(layerInfo.MediaType)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func compressionNameForMediaType(mediaType string) string {
	switch mediaType {
	case imgspecv1.MediaTypeImageLayerGzip,
		manifest.DockerV2Schema2LayerMediaType,
		manifest.DockerV2Schema2ForeignLayerMediaTypeGzip:
		return "gzip"
	case imgspecv1.MediaTypeImageLayerZstd,
		manifest.DockerV2SchemaLayerMediaTypeZstd:
		return "zstd"
	default:
		return ""
	}
}

func manifestLayerInfosToBlobInfos(layerInfos []manifest.LayerInfo) []types.BlobInfo {
	res := make([]types.BlobInfo, len(layerInfos))
	for i, layerInfo := range layerInfos {
		res[i] = layerInfo.BlobInfo
	}
	return res
}

type SimpleConfig struct {
	Architecture string   `json:"architecture,omitempty"`
	OS           string   `json:"os,omitempty"`
	OSVersion    string   `json:"version,omitempty"`
	Variant      string   `json:"variant,omitempty"`
	OSFeatures   []string `json:"features,omitempty"`
}
