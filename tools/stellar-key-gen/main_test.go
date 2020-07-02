package main

import (
	"strings"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_stdoutAndStderr(t *testing.T) {
	cmd := NewRootCmd()
	stdout := strings.Builder{}
	stderr := strings.Builder{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.NoError(t, err)

	// Stdout will be a secret key only.
	f, err := keypair.ParseFull(stdout.String())
	require.NoError(t, err)
	assert.Equal(t, stdout.String(), f.Seed())

	// Stderr will be the public key and some friendly text.
	stderrLines := strings.Split(stderr.String(), "\n")
	require.Len(t, stderrLines, 4)
	assert.Equal(t, "Public Key:", stderrLines[0])
	a, err := keypair.ParseAddress(stderrLines[1])
	require.NoError(t, err)
	assert.Equal(t, stderrLines[1], a.Address())
	assert.Equal(t, "Secret Key:", stderrLines[2])
	assert.Equal(t, "", stderrLines[3])
}

func TestRootCmd_secretKeyRandom(t *testing.T) {
	seenKeys := map[string]bool{}
	for i := 0; i < 10; i++ {
		cmd := NewRootCmd()
		stdout := strings.Builder{}
		stderr := strings.Builder{}
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := cmd.Execute()
		require.NoError(t, err)

		key := stdout.String()
		if seenKeys[key] {
			t.Errorf("%s seen before", key)
			continue
		}
		t.Logf("%s seen first time", key)
		seenKeys[key] = true
	}
}
