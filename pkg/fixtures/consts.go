// SPDX:Apache-2.0
package fixtures

import (
	_ "embed"
)

//go:embed docker-linux-arm64.tar
var DOCKER_LINUX_ARM64 []byte

//go:embed oci-linux-arm64.tar
var OCI_LINUX_ARM64 []byte

//go:embed linux-arm64-tags.txt
var LINUX_ARM64_TAGS string

//go:embed linux-arm64-manifest.json
var LINUX_ARM64_MANIFEST []byte

//go:embed docker-linux-amd64.tar
var DOCKER_LINUX_AMD64 []byte

//go:embed oci-linux-amd64.tar
var OCI_LINUX_AMD64 []byte

//go:embed oci-indexed.tar
var OCI_INDEXED []byte

//go:embed oci-indexed-manifest.json
var OCI_INDEXED_MANIFEST string
