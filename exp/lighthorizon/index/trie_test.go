package index

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"testing"
)

func randomTrie(t *testing.T) (*TrieIndex, map[string]uint32) {
	index := &TrieIndex{}
	inserts := map[string]uint32{}
	numInserts := rand.Intn(100)
	for j := 0; j < numInserts; j++ {
		ledger := uint32(rand.Int63())
		hash := make([]byte, 32)
		if _, err := rand.Read(hash); err != nil {
			t.Error(err.Error())
		}

		inserts[string(hash)] = ledger
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, ledger)
		index.Replace(hash[:], b[:])
	}
	return index, inserts
}

func TestTrieIndex(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		index, inserts := randomTrie(t)

		for key, expected := range inserts {
			hash := base64.URLEncoding.EncodeToString([]byte(key))
			value, ok := index.Get([]byte(key))
			if !ok {
				t.Errorf("Hash not found: %s", hash)
			} else {
				ledger := binary.BigEndian.Uint32(value)
				if ledger != expected {
					t.Errorf("Hash %s found: %v, expected: %v", hash, ledger, expected)
				}
			}
		}
	}
}

func TestTrieIndexReplace(t *testing.T) {
	index := &TrieIndex{}

	key := []byte("key")
	prev, ok := index.Replace(key, []byte("a"))
	if ok || prev != nil {
		t.Errorf("Unexpected previous value: %q, expected: nil", string(prev))
	}

	prev, ok = index.Replace(key, []byte("b"))
	if !ok || string(prev) != "a" {
		t.Errorf("Unexpected previous value: %q, expected: a", string(prev))
	}

	prev, ok = index.Replace(key, []byte("c"))
	if !ok || string(prev) != "b" {
		t.Errorf("Unexpected previous value: %q, expected: b", string(prev))
	}
}

func TestTrieIndexSuffixes(t *testing.T) {
	index := &TrieIndex{}

	prev, ok := index.Replace([]byte("a"), []byte("a"))
	if ok || prev != nil {
		t.Errorf("Unexpected previous value: %q, expected: nil", string(prev))
	}

	prev, ok = index.Replace([]byte("ab"), []byte("ab"))
	if ok || prev != nil {
		t.Errorf("Unexpected previous value: %q, expected: nil", string(prev))
	}

	prev, ok = index.Get([]byte("a"))
	if !ok || string(prev) != "a" {
		t.Errorf("Unexpected previous value: %q, expected: a", string(prev))
	}

	prev, ok = index.Get([]byte("ab"))
	if !ok || string(prev) != "ab" {
		t.Errorf("Unexpected previous value: %q, expected: ab", string(prev))
	}
}

func TestTrieIndexSerialization(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		index, inserts := randomTrie(t)

		// Round-trip it to serialization and back
		buf := &bytes.Buffer{}
		nWritten, err := index.WriteTo(buf)
		if err != nil {
			t.Error(err.Error())
		}

		read := &TrieIndex{}
		nRead, err := read.ReadFrom(buf)
		if err != nil {
			t.Error(err.Error())
		}

		if nWritten != nRead {
			t.Errorf("Wrote %d bytes, but read %d bytes", nWritten, nRead)
		}

		for key, expected := range inserts {
			hash := base64.URLEncoding.EncodeToString([]byte(key))
			value, ok := read.Get([]byte(key))
			if !ok {
				t.Errorf("Hash not found: %s", hash)
			} else {
				ledger := binary.BigEndian.Uint32(value)
				if ledger != expected {
					t.Errorf("Hash %s found: %v, expected: %v", hash, ledger, expected)
				}
			}
		}
	}
}
