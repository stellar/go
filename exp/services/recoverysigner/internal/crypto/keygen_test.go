package crypto

import (
	"strings"
	"testing"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/stretchr/testify/require"
)

func generateHybridKeysetCleartext(t *testing.T) string {
	khPriv, err := keyset.NewHandle(hybrid.ECIESHKDFAES128CTRHMACSHA256KeyTemplate())
	require.NoError(t, err)

	keysetPrivateCleartext := strings.Builder{}
	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	require.NoError(t, err)

	return keysetPrivateCleartext.String()
}

func generateHybridKeysetEncrypted(t *testing.T) string {
	khPriv, err := keyset.NewHandle(hybrid.ECIESHKDFAES128CTRHMACSHA256KeyTemplate())
	require.NoError(t, err)

	keysetPrivateEncrypted := strings.Builder{}
	err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), mockAEAD{})
	require.NoError(t, err)

	return keysetPrivateEncrypted.String()
}
