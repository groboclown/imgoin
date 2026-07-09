// SPDX:Apache-2.0
package imgoin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"

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
	normalized := manifest.NormalizedMIMEType(mime)
	if slices.Contains(manifest.SupportedListMIMETypes, normalized) {
		list, err := manifest.ListFromBlob(raw, mime)
		if err != nil {
			return nil, err
		}
		ret := make([]manifest.ListUpdate, 0)
		for _, digest := range list.Instances() {
			inst, err := list.Instance(digest)
			if err != nil {
				return nil, err
			}
			ret = append(ret, inst)
		}
		return ret, nil
	} else {
		man, err := manifest.FromBlob(raw, mime)
		if err != nil {
			return nil, err
		}
		config := man.ConfigInfo()
		configMan, err := parseConfig(ctx, r.img, config)
		item := manifest.ListUpdate{
			Digest:    config.Digest,
			Size:      config.Size,
			MediaType: config.MediaType,
			ReadOnly: struct {
				Platform                  *imgspecv1.Platform
				Annotations               map[string]string
				CompressionAlgorithmNames []string
				ArtifactType              string
			}{
				Platform: &imgspecv1.Platform{
					Architecture: configMan.Architecture,
					OS:           configMan.OS,
					OSVersion:    configMan.OSVersion,
					OSFeatures:   configMan.OSFeatures,
					Variant:      configMan.Variant,
				},
			},
		}
		item.ReadOnly.Platform = nil
		item.ReadOnly.Annotations = config.Annotations
		if config.CompressionAlgorithm != nil {
			item.ReadOnly.CompressionAlgorithmNames = []string{config.CompressionAlgorithm.Name()}
		}
		item.ReadOnly.ArtifactType = ""

		return []manifest.ListUpdate{item}, nil
	}
}

func (r *SrcImage) GetBlob(ctx context.Context, blob types.BlobInfo) (io.ReadCloser, error) {
	panic("not implemented yet")
}

func (r *SrcImage) GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error) {
	if r.copyContents {
		return nil, nil
	}
	panic("not implemented yet")
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

type SimpleConfig struct {
	Architecture string   `json:"architecture,omitempty"`
	OS           string   `json:"os,omitempty"`
	OSVersion    string   `json:"version,omitempty"`
	Variant      string   `json:"variant,omitempty"`
	OSFeatures   []string `json:"features,omitempty"`
}
