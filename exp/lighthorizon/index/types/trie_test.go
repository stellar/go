package index

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomTrie(t *testing.T, index *TrieIndex) (*TrieIndex, map[string]uint32) {
	if index == nil {
		index = &TrieIndex{}
	}
	inserts := map[string]uint32{}
	numInserts := rand.Intn(100)
	for j := 0; j < numInserts; j++ {
		ledger := uint32(rand.Int63())
		hashBytes := make([]byte, 32)
		if _, err := rand.Read(hashBytes); err != nil {
			assert.NoError(t, err)
		}
		hash := hex.EncodeToString(hashBytes)

		inserts[hash] = ledger
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, ledger)
		index.Upsert([]byte(hash), b)
	}
	return index, inserts
}

func TestTrieIndex(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		index, inserts := randomTrie(t, nil)

		for key, expected := range inserts {
			value, ok := index.Get([]byte(key))
			require.Truef(t, ok, "Key not found: %s", key)
			ledger := binary.BigEndian.Uint32(value)
			assert.Equalf(t, expected, ledger,
				"Key %s found: %v, expected: %v", key, ledger, expected)
		}
	}
}

func TestTrieIndexUpsertBasic(t *testing.T) {
	index := &TrieIndex{}

	key := "key"
	prev, ok := index.Upsert([]byte(key), []byte("a"))
	assert.Nil(t, prev)
	assert.Falsef(t, ok, "expected nil, got prev: %q", string(prev))

	prev, ok = index.Upsert([]byte(key), []byte("b"))
	assert.Equal(t, "a", string(prev))
	assert.Truef(t, ok, "expected 'a', got prev: %q", string(prev))

	prev, ok = index.Upsert([]byte(key), []byte("c"))
	assert.Equal(t, "b", string(prev))
	assert.Truef(t, ok, "expected 'b', got prev: %q", string(prev))
}

func TestTrieIndexSuffixes(t *testing.T) {
	index := &TrieIndex{}

	prev, ok := index.Upsert([]byte("a"), []byte("a"))
	require.False(t, ok)
	require.Nil(t, prev)

	prev, ok = index.Upsert([]byte("ab"), []byte("ab"))
	require.False(t, ok)
	require.Nil(t, prev)

	prev, ok = index.Get([]byte("a"))
	require.True(t, ok)
	require.Equal(t, "a", string(prev))

	prev, ok = index.Get([]byte("ab"))
	require.True(t, ok)
	require.Equal(t, "ab", string(prev))

	prev, ok = index.Upsert([]byte("a"), []byte("b"))
	require.True(t, ok)
	require.Equal(t, "a", string(prev))

	prev, ok = index.Get([]byte("a"))
	require.True(t, ok)
	require.Equal(t, "b", string(prev))
}

func TestTrieIndexSerialization(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			index, inserts := randomTrie(t, nil)

			// Round-trip it to serialization and back
			buf := &bytes.Buffer{}
			nWritten, err := index.WriteTo(buf)
			assert.NoError(t, err)

			read := &TrieIndex{}
			nRead, err := read.ReadFrom(buf)
			assert.NoError(t, err)

			assert.Equal(t, nWritten, nRead, "read more or less than we wrote")

			for key, expected := range inserts {
				value, ok := read.Get([]byte(key))
				require.Truef(t, ok, "Key not found: %s", key)

				ledger := binary.BigEndian.Uint32(value)
				assert.Equal(t, expected, ledger, "for key %s", key)
			}
		})
	}
}

func requireEqualNodes(t *testing.T, expectedNode, gotNode *trieNode) {
	expectedJSON, err := json.Marshal(expectedNode)
	require.NoError(t, err)
	expected := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(expectedJSON, &expected))

	gotJSON, err := json.Marshal(gotNode)
	require.NoError(t, err)
	got := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(gotJSON, &got))

	require.Equal(t, expected, got)
}

func TestTrieIndexUpsertAdvanced(t *testing.T) {
	// TODO: This is janky that we inspect the structure, but I want to make sure
	// I've gotten the algorithms correct.
	makeBase := func() *TrieIndex {
		index := &TrieIndex{}
		index.Upsert([]byte("annibale"), []byte{1})
		index.Upsert([]byte("annibalesco"), []byte{2})
		return index
	}

	t.Run("base", func(t *testing.T) {
		base := makeBase()

		baseExpected := &trieNode{
			Prefix: []byte("annibale"),
			Value:  []byte{1},
			Children: map[byte]*trieNode{
				byte('s'): {
					Prefix: []byte("co"),
					Value:  []byte{2},
				},
			},
		}
		requireEqualNodes(t, baseExpected, base.Root)
	})

	for _, tc := range []struct {
		key      string
		expected *trieNode
	}{
		{"annientare", &trieNode{
			Prefix: []byte("anni"),
			Children: map[byte]*trieNode{
				'b': {
					Prefix: []byte("ale"),
					Value:  []byte{1},
					Children: map[byte]*trieNode{
						's': {
							Prefix: []byte("co"),
							Value:  []byte{2},
						},
					},
				},
				'e': {
					Prefix: []byte("ntare"),
					Value:  []byte{3},
				},
			},
		}},
		{"annibali", &trieNode{
			Prefix: []byte("annibal"),
			Children: map[byte]*trieNode{
				'e': {
					Value: []byte{1},
					Children: map[byte]*trieNode{
						's': {
							Prefix: []byte("co"),
							Value:  []byte{2},
						},
					},
				},
				'i': {
					Value: []byte{3},
				},
			},
		}},
		{"ago", &trieNode{
			Prefix: []byte("a"),
			Children: map[byte]*trieNode{
				'n': {
					Prefix: []byte("nibale"),
					Value:  []byte{1},
					Children: map[byte]*trieNode{
						's': {
							Prefix: []byte("co"),
							Value:  []byte{2},
						},
					},
				},
				'g': {
					Prefix: []byte("o"),
					Value:  []byte{3},
				},
			},
		}},
		{"ciao", &trieNode{
			Children: map[byte]*trieNode{
				'a': {
					Prefix: []byte("nnibale"),
					Value:  []byte{1},
					Children: map[byte]*trieNode{
						's': {
							Prefix: []byte("co"),
							Value:  []byte{2},
						},
					},
				},
				'c': {
					Prefix: []byte("iao"),
					Value:  []byte{3},
				},
			},
		}},
		{"anni", &trieNode{
			Prefix: []byte("anni"),
			Value:  []byte{3},
			Children: map[byte]*trieNode{
				'b': {
					Prefix: []byte("ale"),
					Value:  []byte{1},
					Children: map[byte]*trieNode{
						's': {
							Prefix: []byte("co"),
							Value:  []byte{2},
						},
					},
				},
			},
		}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			// Do our upsert
			index := makeBase()
			index.Upsert([]byte(tc.key), []byte{3})

			// Check the tree is shaped right
			requireEqualNodes(t, tc.expected, index.Root)

			// Check the value matches expected
			value, ok := index.Get([]byte(tc.key))
			require.True(t, ok)
			require.Equal(t, []byte{3}, value)
		})
	}
}

func TestTrieIndexMerge(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		a, aInserts := randomTrie(t, nil)
		b, bInserts := randomTrie(t, nil)

		require.NoError(t, a.Merge(b))

		// Should still have all the A keys
		for key, expected := range aInserts {
			value, ok := a.Get([]byte(key))
			require.Truef(t, ok, "Key not found: %s", key)
			ledger := binary.BigEndian.Uint32(value)
			assert.Equalf(t, expected, ledger, "Key %s found", key)
		}

		// Should now also have all the B keys
		for key, expected := range bInserts {
			value, ok := a.Get([]byte(key))
			require.Truef(t, ok, "Key not found: %s", key)
			ledger := binary.BigEndian.Uint32(value)
			assert.Equalf(t, expected, ledger, "Key %s found", key)
		}
	}
}
