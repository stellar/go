package index

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	TrieIndexVersion = 1

	HeaderHasPrefix   = 0b0000_0001
	HeaderHasValue    = 0b0000_0010
	HeaderHasChildren = 0b0000_0100
)

type TrieIndex struct {
	sync.RWMutex
	Root *trieNode `json:"root"`
}

// TODO: Store the suffix here so we can truncate the branches
type trieNode struct {
	// Common prefix we ignore
	Prefix []byte `json:"prefix,omitempty"`

	// The value of this node.
	Value []byte `json:"value,omitempty"`

	// Any children of this node, mapped by the next byte of their path
	Children map[byte]*trieNode `json:"children,omitempty"`
}

func NewTrieIndexFromBytes(r io.Reader) (*TrieIndex, error) {
	var index TrieIndex
	if _, err := index.ReadFrom(r); err != nil {
		return nil, err
	}
	return &index, nil
}

func (index *TrieIndex) Upsert(key, value []byte) ([]byte, bool) {
	if len(key) == 0 {
		panic("len(key) must be > 0")
	}
	index.Lock()
	defer index.Unlock()
	return index.doUpsert(key, value)
}

func (index *TrieIndex) doUpsert(key, value []byte) ([]byte, bool) {
	if index.Root == nil {
		index.Root = &trieNode{Prefix: key, Value: value}
		return nil, false
	}

	node := index.Root
	var parent *trieNode
	var parentIdx byte
	splitPos := 0
	for len(key) > 0 {
		for splitPos < len(node.Prefix) && len(key) > 0 {
			if node.Prefix[splitPos] != key[0] {
				break
			}
			splitPos++
			key = key[1:]
		}
		if splitPos != len(node.Prefix) {
			// split this node
			break
		}
		if len(key) == 0 {
			// simple update-in-place at this node
			break
		}

		// Jump to the next child
		parent = node
		parentIdx = key[0]
		child, ok := node.Children[key[0]]
		if !ok {
			if node.Children == nil {
				node.Children = map[byte]*trieNode{}
			}
			// child doesn't exist. Insert a new node
			node.Children[key[0]] = &trieNode{
				Prefix: key[1:],
				Value:  value,
			}
			return nil, false
		}
		node = child
		key = key[1:]
		splitPos = 0
	}

	// Key fully consumed just as we reached "node"
	if len(key) == 0 {
		if splitPos == len(node.Prefix) {
			// node prefix matches (or is none), simple update-in-place
			prev := node.Value
			node.Value = value
			return prev, true
		} else {
			// node has a prefix, so we need to insert a new one here and push it down
			splitNode := &trieNode{
				Prefix:   node.Prefix[:splitPos], // the matching segment
				Value:    value,
				Children: map[byte]*trieNode{},
			}
			splitNode.Children[node.Prefix[splitPos]] = node
			node.Prefix = node.Prefix[splitPos+1:] // existing part that didn't match
			if parent == nil {
				index.Root = splitNode
			} else {
				parent.Children[parentIdx] = splitNode
			}
			return nil, false
		}
	} else {
		// leftover key
		if splitPos == len(node.Prefix) {
			// new child
			node.Children[key[0]] = &trieNode{
				Prefix: key[1:],
				Value:  value,
			}
			return nil, false
		} else {
			// Need to split the node
			splitNode := &trieNode{
				Prefix:   node.Prefix[:splitPos],
				Children: map[byte]*trieNode{},
			}
			splitNode.Children[node.Prefix[splitPos]] = node
			splitNode.Children[key[0]] = &trieNode{Prefix: key[1:], Value: value}
			node.Prefix = node.Prefix[splitPos+1:]
			if parent == nil {
				index.Root = splitNode
			} else {
				parent.Children[parentIdx] = splitNode
			}
			return nil, false
		}
	}
}

func (index *TrieIndex) Get(key []byte) ([]byte, bool) {
	index.RLock()
	defer index.RUnlock()
	if index.Root == nil {
		return nil, false
	}

	node := index.Root
	splitPos := 0
	for len(key) > 0 {
		for splitPos < len(node.Prefix) && len(key) > 0 {
			if node.Prefix[splitPos] != key[0] {
				break
			}
			splitPos++
			key = key[1:]
		}
		if splitPos != len(node.Prefix) {
			// split this node
			break
		}
		if len(key) == 0 {
			// found it
			return node.Value, true
		}

		// Jump to the next child
		child, ok := node.Children[key[0]]
		if !ok {
			// child doesn't exist
			return nil, false
		}
		node = child
		key = key[1:]
		splitPos = 0
	}

	if len(key) == 0 {
		return node.Value, true
	}
	return nil, false
}

func (index *TrieIndex) iterate(f func(key, value []byte)) {
	index.RLock()
	defer index.RUnlock()
	if index.Root != nil {
		index.Root.iterate(nil, f)
	}
}

func (node *trieNode) iterate(prefix []byte, f func(key, value []byte)) {
	key := append(prefix, node.Prefix...)
	if len(node.Value) > 0 {
		f(key, node.Value)
	}

	if node.Children != nil {
		for b, child := range node.Children {
			child.iterate(append(key, b), f)
		}
	}
}

// TODO: For now this ignores duplicates. should it error?
func (i *TrieIndex) Merge(other *TrieIndex) error {
	i.Lock()
	defer i.Unlock()

	other.iterate(func(key, value []byte) {
		i.doUpsert(key, value)
	})

	return nil
}

// TODO: Use XDR for this, to be more consistent with rest of the codebase, and
// do less custom shenanigans.
func (i *TrieIndex) ReadFrom(r io.Reader) (int64, error) {
	i.Lock()
	defer i.Unlock()

	var nRead int64
	br := bufio.NewReader(r)

	// Read the index version
	version, err := binary.ReadUvarint(br)
	nRead += int64(uvarintSize(version))
	if err != nil {
		return nRead, err
	} else if version != TrieIndexVersion {
		return nRead, fmt.Errorf("unsupported trie version: %d", version)
	}

	i.Root = &trieNode{}
	n, err := i.Root.readFrom(br)
	return nRead + n, err
}

func (i *trieNode) readFrom(r *bufio.Reader) (int64, error) {
	var nRead int64

	// Read the header flags byte
	header, err := r.ReadByte()
	nRead += 1
	if err != nil {
		return nRead, err
	}

	// Read this node's prefix
	if header&HeaderHasPrefix > 0 {
		prefix, n64, err := readBytes(r)
		nRead += n64
		if err != nil {
			return nRead, err
		}
		i.Prefix = prefix
	}

	// Read this node's value
	if header&HeaderHasValue > 0 {
		value, n64, err := readBytes(r)
		nRead += n64
		if err != nil {
			return nRead, err
		}
		i.Value = value
	}

	// Read this node's children count
	if header&HeaderHasChildren > 0 {
		childLen, err := binary.ReadUvarint(r)
		nRead += int64(uvarintSize(childLen))
		if err != nil {
			return nRead, err
		}

		if childLen > 0 {
			i.Children = map[byte]*trieNode{}
			// Read this node's children
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
				i.Children[key] = &node
			}
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
	i.RLock()
	defer i.RUnlock()
	buf := make([]byte, binary.MaxVarintLen64)

	var nWritten, n64 int64

	// Write the index version
	n := binary.PutUvarint(buf, uint64(TrieIndexVersion))
	n, err := w.Write(buf[:n])
	nWritten += int64(n)
	if err != nil {
		return nWritten, err
	}

	if i.Root == nil {
		n64, err = (&trieNode{}).writeTo(w, buf)
	} else {
		n64, err = i.Root.writeTo(w, buf)
	}
	return nWritten + n64, err
}

func (i *trieNode) writeTo(w io.Writer, buf []byte) (int64, error) {
	var nWritten, n64 int64

	// Write the header flags byte
	var header byte
	if len(i.Prefix) > 0 {
		header |= HeaderHasPrefix
	}
	if len(i.Value) > 0 {
		header |= HeaderHasValue
	}
	if i.Children != nil && len(i.Children) > 0 {
		header |= HeaderHasChildren
	}
	n, err := w.Write([]byte{header})
	nWritten += int64(n)
	if err != nil {
		return nWritten, err
	}

	// Write this node's prefix
	if header&HeaderHasPrefix > 0 {
		n64, err := writeBytes(w, i.Prefix, buf)
		nWritten += n64
		if err != nil {
			return nWritten, err
		}
	}

	// Write this node's value
	if header&HeaderHasValue > 0 {
		n64, err = writeBytes(w, i.Value, buf)
		nWritten += n64
		if err != nil {
			return nWritten, err
		}
	}

	// TODO: Can we write an "index" of sorts, here that has the byte-offsets, so
	// that we do just-in-time parsing? Might be more verbose than as is, tho

	// Write how many children we have
	if header&HeaderHasChildren > 0 {
		n = binary.PutUvarint(buf, uint64(len(i.Children)))
		n, err = w.Write(buf[:n])
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}

		// Write all the children
		for key, child := range i.Children {
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
	}

	return nWritten, nil
}

// Read a length-prefixed chunk of bytes
func readBytes(r *bufio.Reader) ([]byte, int64, error) {
	var nRead int64
	// Read this node's value's length
	valueLen, err := binary.ReadUvarint(r)
	nRead += int64(uvarintSize(valueLen))
	if err != nil || valueLen == 0 {
		return nil, nRead, err
	}

	// Read this node's value
	data := make([]byte, valueLen)
	n, err := io.ReadFull(r, data)
	nRead += int64(n)
	if err != nil {
		return nil, nRead, err
	}
	return data, nRead, nil
}

// Write a length-prefixed chunk of bytes
func writeBytes(w io.Writer, data, scratch []byte) (int64, error) {
	var nWritten int64
	n := binary.PutUvarint(scratch, uint64(len(data)))
	n, err := w.Write(scratch[:n])
	nWritten += int64(n)
	if err != nil || len(data) == 0 {
		return nWritten, err
	}

	n, err = w.Write(data)
	return nWritten + int64(n), err
}
