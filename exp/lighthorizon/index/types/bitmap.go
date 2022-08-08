package index

import (
	"bytes"
	"io"
	"sync"

	"github.com/stellar/go/xdr"
)

const BitmapIndexVersion = 1

type BitmapIndex struct {
	mutex    sync.RWMutex
	bitmap   []byte
	firstBit uint32
	lastBit  uint32
}

type NamedIndices map[string]*BitmapIndex

func NewBitmapIndex(b []byte) (*BitmapIndex, error) {
	xdrBitmap := xdr.BitmapIndex{}
	err := xdrBitmap.UnmarshalBinary(b)
	if err != nil {
		return nil, err
	}

	return NewBitmapIndexFromXDR(xdrBitmap), nil
}

func NewBitmapIndexFromXDR(index xdr.BitmapIndex) *BitmapIndex {
	return &BitmapIndex{
		bitmap:   index.Bitmap[:],
		firstBit: uint32(index.FirstBit),
		lastBit:  uint32(index.LastBit),
	}
}

func (i *BitmapIndex) Size() int {
	return len(i.bitmap)
}

func (i *BitmapIndex) SetActive(index uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.setActive(index)
}

func bitShiftLeft(index uint32) byte {
	if index%8 == 0 {
		return 1
	} else {
		return byte(1) << (8 - index%8)
	}
}

func (i *BitmapIndex) rangeFirstBit() uint32 {
	return (i.firstBit-1)/8*8 + 1
}

func (i *BitmapIndex) rangeLastBit() uint32 {
	return i.rangeFirstBit() + uint32(len(i.bitmap))*8 - 1
}

func (i *BitmapIndex) setActive(index uint32) error {
	if i.firstBit == 0 {
		i.firstBit = index
		i.lastBit = index
		b := bitShiftLeft(index)
		i.bitmap = []byte{b}
	} else {
		if index >= i.rangeFirstBit() && index <= i.rangeLastBit() {
			// Update the bit in existing range
			b := bitShiftLeft(index)
			loc := (index - i.rangeFirstBit()) / 8
			i.bitmap[loc] = i.bitmap[loc] | b

			if index < i.firstBit {
				i.firstBit = index
			}
			if index > i.lastBit {
				i.lastBit = index
			}
		} else {
			// Expand the bitmap
			if index < i.rangeFirstBit() {
				// ...to the left
				c := (i.rangeFirstBit() - index) / 8
				if (i.rangeFirstBit()-index)%8 != 0 {
					c++
				}
				newBytes := make([]byte, c)
				i.bitmap = append(newBytes, i.bitmap...)

				b := bitShiftLeft(index)
				i.bitmap[0] = i.bitmap[0] | b

				i.firstBit = index
			} else if index > i.rangeLastBit() {
				// ... to the right
				newBytes := make([]byte, (index-i.rangeLastBit())/8+1)
				i.bitmap = append(i.bitmap, newBytes...)
				b := bitShiftLeft(index)
				loc := (index - i.rangeFirstBit()) / 8
				i.bitmap[loc] = i.bitmap[loc] | b

				i.lastBit = index
			}
		}
	}

	return nil
}

//lint:ignore U1000 Ignore unused function temporarily
func (i *BitmapIndex) isActive(index uint32) bool {
	if index >= i.firstBit && index <= i.lastBit {
		b := bitShiftLeft(index)
		loc := (index - i.rangeFirstBit()) / 8
		return i.bitmap[loc]&b != 0
	} else {
		return false
	}
}

func (i *BitmapIndex) iterate(f func(index uint32)) error {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	if i.firstBit == 0 {
		return nil
	}

	f(i.firstBit)
	curr := i.firstBit

	for {
		var err error
		curr, err = i.nextActiveBit(curr + 1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		f(curr)
	}

	return nil
}

func (i *BitmapIndex) Merge(other *BitmapIndex) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	var err error
	other.iterate(func(index uint32) {
		if err != nil {
			return
		}
		err = i.setActive(index)
	})

	return err
}

// NextActiveBit returns the next bit position (inclusive) where this index is
// active. "Inclusive" means that if it's already active at `position`, this
// returns `position`.
func (i *BitmapIndex) NextActiveBit(position uint32) (uint32, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.nextActiveBit(position)
}

func (i *BitmapIndex) nextActiveBit(position uint32) (uint32, error) {
	if i.firstBit == 0 || position > i.lastBit {
		// We're past the end.
		// TODO: Should this be an error? or how should we signal NONE here?
		return 0, io.EOF
	}

	if position < i.firstBit {
		position = i.firstBit
	}

	// Must be within the range, find the first non-zero after our start
	loc := (position - i.rangeFirstBit()) / 8

	// Is it in the same byte?
	if shift, ok := maxBitAfter(i.bitmap[loc], (position-1)%8); ok {
		return i.rangeFirstBit() + (loc * 8) + shift, nil
	}

	// Scan bytes after
	loc++
	for ; loc < uint32(len(i.bitmap)); loc++ {
		// Find the offset of the set bit
		if shift, ok := maxBitAfter(i.bitmap[loc], 0); ok {
			return i.rangeFirstBit() + (loc * 8) + shift, nil
		}
	}

	// all bits after this were zero
	// TODO: Should this be an error? or how should we signal NONE here?
	return 0, io.EOF
}

func maxBitAfter(b byte, after uint32) (uint32, bool) {
	if b == 0 {
		// empty byte
		return 0, false
	}

	for shift := uint32(after); shift < 8; shift++ {
		mask := byte(0b1000_0000) >> shift
		if mask&b != 0 {
			return shift, true
		}
	}
	return 0, false
}

func (i *BitmapIndex) ToXDR() xdr.BitmapIndex {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return xdr.BitmapIndex{
		FirstBit: xdr.Uint32(i.firstBit),
		LastBit:  xdr.Uint32(i.lastBit),
		Bitmap:   i.bitmap,
	}
}

func (i *BitmapIndex) Buffer() *bytes.Buffer {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	xdrBitmap := i.ToXDR()
	b, err := xdrBitmap.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(b)
}

// Flush flushes the index data to byte slice in index format.
func (i *BitmapIndex) Flush() []byte {
	return i.Buffer().Bytes()
}
