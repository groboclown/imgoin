// SPDX:Apache-2.0

package imgoin

// Tests to ensure the protonmail crypto library is installed to allow a work-around for
// the OpenPGP support issues.

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
)

func Test_EnsureOpenPgp(t *testing.T) {
	keyring := openpgp.EntityList{}
	md, err := openpgp.ReadMessage(bytes.NewReader([]byte{}), keyring, nil, nil)
	if err == nil {
		// Invalid key, invalid message.
		t.Fatal(err)
	}
	if md != nil {
		t.Fatal(md)
	}
}
