package index

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const TrieIndexVersion = 1

type TrieIndex struct {
	mutex sync.RWMutex
	trieNode
}

// TODO: Store the suffix here so we can truncate the branches
type trieNode struct {
	value []byte

	children map[byte]*trieNode
}

func NewTrieIndexFromBytes(r io.Reader) (*TrieIndex, error) {
	return nil, fmt.Errorf("TODO: Implement NewTrieIndexFromReader")
}

func (i *TrieIndex) Replace(key, value []byte) ([]byte, bool) {
	if len(key) == 0 {
		panic("len(key) must be > 0")
	}
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.replace(key, value)
}

// TODO: Not sure this is right..
func (i *trieNode) replace(key, value []byte) ([]byte, bool) {
	n := i
	for j := 0; j < len(key); j++ {
		if n.children == nil {
			n.children = map[byte]*trieNode{}
		}
		child, exists := n.children[key[j]]
		if !exists {
			child = &trieNode{}
			n.children[key[j]] = child
		}
		n = child
	}
	prev := n.value
	n.value = value
	return prev, prev != nil
}

func (i *TrieIndex) Get(key []byte) ([]byte, bool) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.get(key)
}

func (i *trieNode) get(key []byte) ([]byte, bool) {
	n := i
	for j := 0; j < len(key); j++ {
		if n.children == nil {
			return nil, false
		}
		child, exists := n.children[key[j]]
		if !exists {
			return nil, false
		}
		n = child
	}
	return n.value, true
}

func (i *TrieIndex) Merge(other *TrieIndex) error {
	return fmt.Errorf("TODO: Implement TrieIndex.Merge")
}

// TODO: Use XDR for this, to be more consistent with rest of the codebase, and
// do less custom shenanigans.
func (i *TrieIndex) ReadFrom(r io.Reader) (int64, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.readFrom(bufio.NewReader(r))
}

func (i *trieNode) readFrom(r *bufio.Reader) (int64, error) {
	var nRead int64

	// Read this node's value's length
	valueLen, err := binary.ReadUvarint(r)
	nRead += int64(uvarintSize(valueLen))
	if err != nil {
		return nRead, err
	}

	if valueLen > 0 {
		// Read this node's value
		i.value = make([]byte, valueLen)
		n, err := io.ReadFull(r, i.value)
		nRead += int64(n)
		if err != nil {
			return nRead, err
		}
	}

	// Read this node's children count
	childLen, err := binary.ReadUvarint(r)
	nRead += int64(uvarintSize(childLen))
	if err != nil {
		return nRead, err
	}

	if childLen > 0 {
		// Read this node's children
		i.children = map[byte]*trieNode{}
		for j := uint64(0); j < childLen; j++ {
			// Read the child's key
			key, err := r.ReadByte()
			nRead += 1
			if err != nil {
				return nRead, err
			}

			// Read the rest of the child
			var node trieNode
			n64, err := node.readFrom(r)
			nRead += n64
			if err != nil {
				return nRead, err
			}
			i.children[key] = &node
		}
	}

	return nRead, nil
}

// TODO: Do this better, without allocating a new byte buffer each time, etc..
func uvarintSize(value uint64) int {
	return binary.PutUvarint(make([]byte, binary.MaxVarintLen64), value)
}

// TODO: Use XDR for this, to be more consistent with rest of the codebase, and
// do less custom shenanigans.
func (i *TrieIndex) WriteTo(w io.Writer) (int64, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.writeTo(w, make([]byte, binary.MaxVarintLen64))
}

func (i *trieNode) writeTo(w io.Writer, buf []byte) (int64, error) {
	var nWritten int64

	// Write the length of this node's value
	n := binary.PutUvarint(buf, uint64(len(i.value)))
	n, err := w.Write(buf[:n])
	nWritten += int64(n)
	if err != nil {
		return nWritten, err
	}

	// Write this node's value
	if len(i.value) > 0 {
		n, err := w.Write(i.value)
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}
	}

	// TODO: Can we write an "index" of sorts, here that has the byte-offsets, so
	// that we do just-in-time parsing? Might be more verbose than as is, tho

	// Write how many children we have
	n = binary.PutUvarint(buf, uint64(len(i.children)))
	n, err = w.Write(buf[:n])
	nWritten += int64(n)
	if err != nil {
		return nWritten, err
	}

	// Write all the children
	for key, child := range i.children {
		// Write the child's key
		n, err = w.Write([]byte{key})
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}

		// Write the rest of the child
		n64, err := child.writeTo(w, buf)
		nWritten += n64
		if err != nil {
			return nWritten, err
		}
	}

	return nWritten, nil
}
