// SPDX:Apache-2.0
package cmd_test

import (
	"io"
	"os"
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

// TestBadArgument ensures that a bad argument generates an error response.
func TestBadArgument(t *testing.T) {
	parsed, res := cmd.ParseArgs("test-proc", "1", []string{"--tuna"})
	if res != 1 {
		t.Errorf("expected exit code 1, found %v", res)
	}
	if parsed != nil {
		t.Errorf("expected nil parsed value, found %v", parsed)
	}
}

// TestVersionHardCodesName ensures that the program name is hard-coded.
// Regardless of what the user links or copies the program name to, this
// allows tools to figure out the actual tool name.
func TestVersionHardCodesName(t *testing.T) {
	parsed, res, out := captureParseStdOut("guppy", "1.2.3", []string{"-V"})

	if res != 0 {
		t.Errorf("expected exit code 0, found %v", res)
	}
	if parsed != nil {
		t.Errorf("expected nil parsed value, found %v", parsed)
	}
	if out != "imgoin v1.2.3\n" {
		t.Errorf("expected 'imgoin v1.2.3\n', found '%v'", out)
	}
}

func captureParseStdOut(name, version string, args []string) (*cmd.ParsedArgs, int, string) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	parsed, res := cmd.ParseArgs(name, version, args)
	os.Stdout = orig
	w.Close()
	out, _ := io.ReadAll(r)
	return parsed, res, string(out)
}
