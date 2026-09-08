#!/usr/bin/env bash
# Taken from buildah:
# https://github.com/podman-container-tools/buildah/blob/main/btrfs_installed_tag.sh
${CC} ${CFLAGS} - > /dev/null 2> /dev/null << EOF
#include <btrfs/ioctl.h>
EOF
if test $? -ne 0 ; then
	echo exclude_graphdriver_btrfs
fi
