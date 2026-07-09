// SPDX:Apache-2.0

package imgoin

import (
	"context"
	"io"

	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/types"
)

var _ TargetImage = (*StdTgtImage)(nil)

func (b *BearingImage) AsTargetImage(ctx context.Context) (*StdTgtImage, error) {
	img, err := b.ref.NewImageDestination(ctx, b.sys)
	if err != nil {
		return nil, err
	}
	return &StdTgtImage{img}, nil
}

type StdTgtImage struct {
	img types.ImageDestination
}

func (s *StdTgtImage) Close() error {
	return s.img.Close()
}

func (s *StdTgtImage) AddManifest(_lu manifest.ListUpdate) error {
	panic("not implemented")
}

func (s *StdTgtImage) CopyBlob(_bi types.BlobInfo, rc io.ReadCloser) error {
	defer rc.Close()
	panic("not implemented")
}
