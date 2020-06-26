package crypto

import (
	"strings"
	"testing"

	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"
	"github.com/stretchr/testify/require"
)

func generateKeysetCleartext(t *testing.T, keyTemplate *tinkpb.KeyTemplate) string {
	khPriv, err := keyset.NewHandle(keyTemplate)
	require.NoError(t, err)

	keysetPrivateCleartext := strings.Builder{}
	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	require.NoError(t, err)

	return keysetPrivateCleartext.String()
}

func generateKeysetEncrypted(t *testing.T, keyTemplate *tinkpb.KeyTemplate) string {
	khPriv, err := keyset.NewHandle(keyTemplate)
	require.NoError(t, err)

	keysetPrivateEncrypted := strings.Builder{}
	err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), mockAEAD{})
	require.NoError(t, err)

	return keysetPrivateEncrypted.String()
}

func rotateKeysetCleartext(t *testing.T, keysetCleartextJSON string, keyTemplate *tinkpb.KeyTemplate) string {
	khPriv, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(keysetCleartextJSON)))
	require.NoError(t, err)

	m := keyset.NewManagerFromHandle(khPriv)
	err = m.Rotate(keyTemplate)
	require.NoError(t, err)

	khPriv, err = m.Handle()
	require.NoError(t, err)

	keysetPrivateCleartext := strings.Builder{}
	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	require.NoError(t, err)

	return keysetPrivateCleartext.String()
}

func rotateKeysetEncrypted(t *testing.T, keysetEncryptedJSON string, keyTemplate *tinkpb.KeyTemplate) string {
	khPriv, err := keyset.Read(keyset.NewJSONReader(strings.NewReader(keysetEncryptedJSON)), mockAEAD{})
	require.NoError(t, err)

	m := keyset.NewManagerFromHandle(khPriv)
	err = m.Rotate(keyTemplate)
	require.NoError(t, err)

	khPriv, err = m.Handle()
	require.NoError(t, err)

	keysetPrivateEncrypted := strings.Builder{}
	err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), mockAEAD{})
	require.NoError(t, err)

	return keysetPrivateEncrypted.String()
}
