// SPDX:Apache-2.0
//go:build linux

package cmd

import (
	"fmt"
	"slices"

	"github.com/moby/sys/capability"
	"go.podman.io/storage/pkg/unshare"
)

var neededCapabilities = []capability.Cap{
	capability.CAP_CHOWN,
	capability.CAP_DAC_OVERRIDE,
	capability.CAP_FOWNER,
	capability.CAP_FSETID,
	capability.CAP_MKNOD,
	capability.CAP_SETFCAP,
	capability.CAP_SYS_ADMIN,
}

func MaybeReexec() error {
	capabilities, err := capability.NewPid2(0)
	if err != nil {
		return fmt.Errorf("error reading the current capabilities sets: %w", err)
	}
	if err := capabilities.Load(); err != nil {
		return fmt.Errorf("error loading the current capabilities sets: %w", err)
	}
	if slices.ContainsFunc(neededCapabilities, func(cap capability.Cap) bool {
		return !capabilities.Get(capability.EFFECTIVE, cap)
	}) {
		unshare.MaybeReexecUsingUserNamespace(true)
		return nil
	}
	return nil
}
