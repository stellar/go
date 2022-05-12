package index

import (
	"bufio"
	"encoding"
	"io"
	"sync"

	"github.com/stellar/go/exp/lighthorizon/index/xdr"
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

func (i *TrieIndex) MarshalBinary() ([]byte, error) {
	i.RLock()
	defer i.RUnlock()

	xdrRoot := xdr.TrieNode{}

	// Apparently this is possible?
	if i.Root != nil {
		xdrRoot.Prefix = i.Root.Prefix
		xdrRoot.Value = i.Root.Value
		xdrRoot.Children = make([]xdr.TrieNodeChild, 0, len(i.Root.Children))

		for key, node := range i.Root.Children {
			buildXdrTrie(key, node, &xdrRoot)
		}
	}

	xdrIndex := xdr.TrieIndex{Version: TrieIndexVersion, Root: xdrRoot}
	return xdrIndex.MarshalBinary()
}

func (i *TrieIndex) WriteTo(w io.Writer) (int64, error) {
	i.RLock()
	defer i.RUnlock()

	bytes, err := i.MarshalBinary()
	if err != nil {
		return int64(len(bytes)), err
	}

	count, err := w.Write(bytes)
	return int64(count), err
}

func (i *TrieIndex) UnmarshalBinary(bytes []byte) error {
	i.RLock()
	defer i.RUnlock()

	xdrIndex := xdr.TrieIndex{}
	err := xdrIndex.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	i.Root = &trieNode{
		Prefix:   xdrIndex.Root.Prefix,
		Value:    xdrIndex.Root.Value,
		Children: make(map[byte]*trieNode, len(xdrIndex.Root.Children)),
	}

	for _, node := range xdrIndex.Root.Children {
		buildTrie(&node, i.Root)
	}

	return nil
}

func (i *TrieIndex) ReadFrom(r io.Reader) (int64, error) {
	i.RLock()
	defer i.RUnlock()

	br := bufio.NewReader(r)
	bytes, err := io.ReadAll(br)
	if err != nil {
		return int64(len(bytes)), err
	}

	return int64(len(bytes)), i.UnmarshalBinary(bytes)
}

// buildTrie recursively builds the equivalent `TrieNode` structure from raw
// XDR, creating the key->value child mapping from the flat list of children.
// Here, `xdrNode` is the node we're processing and `parent` is its non-XDR
// parent (i.e. the parent was already converted from XDR).
//
// This is the opposite of buildXdrTrie.
func buildTrie(xdrNode *xdr.TrieNodeChild, parent *trieNode) {
	node := &trieNode{
		Prefix:   xdrNode.Node.Prefix,
		Value:    xdrNode.Node.Value,
		Children: make(map[byte]*trieNode, len(xdrNode.Node.Children)),
	}
	parent.Children[xdrNode.Key[0]] = node

	for _, child := range xdrNode.Node.Children {
		buildTrie(&child, node)
	}
}

// buildXdrTrie recursively builds the XDR-equivalent TrieNode structure, where
// `i` is the node we're converting and `parent` is the already-converted
// parent. That is, the non-XDR version of `parent` should have had (`key`, `i`)
// as a child.
//
// This is the opposite of buildTrie.
func buildXdrTrie(key byte, node *trieNode, parent *xdr.TrieNode) {
	self := xdr.TrieNode{
		Prefix:   node.Prefix,
		Value:    node.Value,
		Children: make([]xdr.TrieNodeChild, 0, len(node.Children)),
	}

	for key, node := range node.Children {
		buildXdrTrie(key, node, &self)
	}

	parent.Children = append(parent.Children, xdr.TrieNodeChild{
		Key:  [1]byte{key},
		Node: self,
	})
}

// Ensure we're compatible with stdlib interfaces.
var _ io.WriterTo = &TrieIndex{}
var _ io.ReaderFrom = &TrieIndex{}

var _ encoding.BinaryMarshaler = &TrieIndex{}
var _ encoding.BinaryUnmarshaler = &TrieIndex{}
