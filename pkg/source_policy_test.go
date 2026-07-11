// SPDX:Apache-2.0
package imgoin_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	imgoin "github.com/groboclown/imgoin/pkg"
	"github.com/groboclown/imgoin/pkg/fixtures"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/types"
)

func TestAsSourceImageSignaturePolicy(t *testing.T) {
	tmpdir := t.TempDir()
	tarFile := path.Join(tmpdir, "oci.tar")
	if err := os.WriteFile(tarFile, fixtures.OCI_LINUX_AMD64, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	const rejectJSON = `{"default":[{"type":"reject"}]}`
	const insecureJSON = `{"default":[{"type":"insecureAcceptAnything"}]}`
	const secureJSON = `{"default":[{"type":"sigstoreSigned", "keyData": ""}]}`

	policyFile := path.Join(tmpdir, "policy.json")

	tests := []struct {
		name       string
		reqSign    bool
		policyData string
		wantErr    bool
	}{
		{
			name:       "reject policy",
			reqSign:    false,
			policyData: rejectJSON,
			wantErr:    true,
		},
		{
			name:       "insecure bypass",
			reqSign:    false,
			policyData: "",
			wantErr:    false,
		},
		{
			name:       "require signed insecure",
			reqSign:    false,
			policyData: insecureJSON,
			wantErr:    false,
		},
		{
			name:       "require signed secure",
			reqSign:    false,
			policyData: secureJSON,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := &types.SystemContext{}
			var policy *signature.Policy = nil
			if tt.policyData != "" {
				if err := os.WriteFile(policyFile, []byte(tt.policyData), 0o600); err != nil {
					t.Fatal(err)
				}
				sys.SignaturePolicyPath = policyFile
				p, err := signature.DefaultPolicy(sys)
				if err != nil {
					t.Fatal(err)
				}
				policy = p
			}

			img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
				ImageUri:         fmt.Sprintf("oci-archive:%s", tarFile),
				System:           &types.SystemContext{},
				Policy:           policy,
				RequireSignature: tt.reqSign,
			})
			if err != nil {
				t.Fatal(err)
			}

			src, err := img.AsSourceImage(t.Context(), false)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected source image policy validation to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer src.Close()

			manifests, err := src.GetManifests(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if len(manifests) != 1 {
				t.Fatalf("wrong number of manifests: %d, expected 1", len(manifests))
			}
		})
	}
}
