package index

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/stellar/go/support/ordered"
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

func (i *BitmapIndex) SetInactive(index uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.setInactive(index)
}

// bitShiftLeft returns a byte with the bit set corresponding to the index. In
// other words, it flips the bit corresponding to the index's "position" mod-8.
func bitShiftLeft(index uint32) byte {
	if index%8 == 0 {
		return 1
	} else {
		return byte(1) << (8 - index%8)
	}
}

// rangeFirstBit returns the index of the first *possible* active bit in the
// bitmap. In other words, if you just have SetActive(12), this will return 9,
// because you have one byte (0b0001_0000) and the *first* value the bitmap can
// represent is 9.
func (i *BitmapIndex) rangeFirstBit() uint32 {
	return (i.firstBit-1)/8*8 + 1
}

// rangeLastBit returns the index of the last *possible* active bit in the
// bitmap. In other words, if you just have SetActive(12), this will return 16,
// because you have one byte (0b0001_0000) and the *last* value the bitmap can
// represent is 16.
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
				newBytes := make([]byte, distance(index, i.rangeFirstBit()))
				i.bitmap = append(newBytes, i.bitmap...)
				b := bitShiftLeft(index)
				i.bitmap[0] = i.bitmap[0] | b

				i.firstBit = index
			} else if index > i.rangeLastBit() {
				// ... to the right
				newBytes := make([]byte, distance(i.rangeLastBit(), index))
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

func (i *BitmapIndex) setInactive(index uint32) error {
	// Is this index even active in the first place?
	if i.firstBit == 0 || index < i.rangeFirstBit() || index > i.rangeLastBit() {
		return nil // not really an error
	}

	loc := (index - i.rangeFirstBit()) / 8 // which byte?
	b := bitShiftLeft(index)               // which bit w/in the byte?
	i.bitmap[loc] &= ^b                    // unset only that bit

	// If unsetting this bit made the first byte empty OR we unset the earliest
	// set bit, we need to find the next "first" active bit.
	if loc == 0 && i.firstBit == index {
		// find the next active bit to set as the start
		nextBit, err := i.nextActiveBit(index)
		if err == io.EOF {
			i.firstBit = 0
			i.lastBit = 0
			i.bitmap = []byte{}
		} else if err != nil {
			return err
		} else {
			// Trim all (now-)empty bytes off the front.
			i.bitmap = i.bitmap[distance(i.firstBit, nextBit):]
			i.firstBit = nextBit
		}
	} else if int(loc) == len(i.bitmap)-1 {
		idx := -1

		if i.bitmap[loc] == 0 {
			// find the latest non-empty byte, to set as the new "end"
			j := len(i.bitmap) - 1
			for i.bitmap[j] == 0 {
				j--
			}

			i.bitmap = i.bitmap[:j+1]
			idx = 8
		} else if i.lastBit == index {
			// Get the "bit number" of the last active bit (i.e. the one we just
			// turned off) to mark the starting point for the search.
			idx = 8
			if index%8 != 0 {
				idx = int(index % 8)
			}
		}

		// Do we need to adjust the range? Imagine we had 0b0011_0100 and we
		// unset the last active bit.
		//         ^
		// Then, we need to adjust our internal lastBit tracker to represent the
		// ^ bit above. This means finding the first previous set bit.
		if idx > -1 {
			l := uint32(len(i.bitmap) - 1)
			// Imagine we had 0b0011_0100 and we unset the last active bit.
			//                     ^
			// Then, we need to adjust our internal lastBit tracker to represent
			// the ^ bit above. This means finding the first previous set bit.
			j, ok := int(idx), false
			for ; j >= 0 && !ok; j-- {
				_, ok = maxBitAfter(i.bitmap[l], uint32(j))
			}

			// We know from the earlier conditional that *some* bit is set, so
			// we know that j represents the index of the bit that's the new
			// "last active" bit.
			firstByte := i.rangeFirstBit()
			i.lastBit = firstByte + (l * 8) + uint32(j) + 1
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

// DebugCompare returns a string that compares this bitmap to another bitmap
// byte-by-byte in binary form as two columns.
func (i *BitmapIndex) DebugCompare(j *BitmapIndex) string {
	output := make([]string, ordered.Max(len(i.bitmap), len(j.bitmap)))
	for n := 0; n < len(output); n++ {
		if n < len(i.bitmap) {
			output[n] += fmt.Sprintf("%08b", i.bitmap[n])
		} else {
			output[n] += "        "
		}

		output[n] += " | "

		if n < len(j.bitmap) {
			output[n] += fmt.Sprintf("%08b", j.bitmap[n])
		}
	}

	return strings.Join(output, "\n")
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

// distance returns how many bytes occur between the two given indices. Note
// that j >= i, otherwise the result will be negative.
func distance(i, j uint32) int {
	return (int(j)-1)/8 - (int(i)-1)/8
}
