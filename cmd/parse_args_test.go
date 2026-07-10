// SPDX:Apache-2.0
package cmd_test

import (
	"testing"

	"github.com/groboclown/imgoin/cmd"
)

func TestTransportRequireRootless(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{
			name:  "containers storage source",
			image: "containers-storage:localhost/local-alpine:linux-amd64",
			want:  true,
		},
		{
			name:  "containers storage destination",
			image: "oci-archive:/tmp/x.tar",
			want:  false,
		},
		{
			name:  "non storage images",
			image: "docker://quay.io/skopeo/stable:latest",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if found := cmd.TransportRequireRootless(tt.image); found != tt.want {
				t.Fatalf("imageNamesRequireRootlessReexec() = %v, want %v", found, tt.want)
			}
		})
	}
}
