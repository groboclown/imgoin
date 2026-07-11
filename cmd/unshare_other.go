// SPDX:Apache-2.0
//go:build !linux

package cmd

func MaybeReexec() error {
	return nil
}
