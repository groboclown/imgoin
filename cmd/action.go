// SPDX:Apache-2.0
package cmd

import (
	"errors"

	imgoin "github.com/groboclown/imgoin/pkg"
)

type Action struct {
	Sources []imgoin.SourceReference
	Target  imgoin.TargetImage
}

func (p *Action) Close() error {
	errs := make([]error, 0)
	for _, s := range p.Sources {
		errs = append(errs, s.Close())
	}
	errs = append(errs, p.Target.Close())
	return errors.Join(errs...)
}
