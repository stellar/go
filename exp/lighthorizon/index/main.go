package index

import (
	"bytes"
	"io"
	"sync"

	"github.com/stellar/go/exp/lighthorizon/index/xdr"
)

const CheckpointIndexVersion = 1

type CheckpointIndex struct {
	mutex           sync.RWMutex
	bitmap          []byte
	firstCheckpoint uint32
	lastCheckpoint  uint32
}

func NewCheckpointIndexFromBytes(b []byte) (*CheckpointIndex, error) {
	xdrCheckpoint := xdr.CheckpointIndex{}
	err := xdrCheckpoint.UnmarshalBinary(b)
	if err != nil {
		return nil, err
	}

	return &CheckpointIndex{
		bitmap:          xdrCheckpoint.Bitmap,
		firstCheckpoint: uint32(xdrCheckpoint.FirstCheckpoint),
		lastCheckpoint:  uint32(xdrCheckpoint.LastCheckpoint),
	}, nil
}

func (i *CheckpointIndex) SetActive(checkpoint uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.setActive(checkpoint)
}

func bitShiftLeft(checkpoint uint32) byte {
	if checkpoint%8 == 0 {
		return 1
	} else {
		return byte(1) << (8 - checkpoint%8)
	}
}

func (i *CheckpointIndex) rangeFirstCheckpoint() uint32 {
	return (i.firstCheckpoint-1)/8*8 + 1
}

func (i *CheckpointIndex) rangeLastCheckpoint() uint32 {
	return i.rangeFirstCheckpoint() + uint32(len(i.bitmap))*8 - 1
}

func (i *CheckpointIndex) setActive(checkpoint uint32) error {
	if i.firstCheckpoint == 0 {
		i.firstCheckpoint = checkpoint
		i.lastCheckpoint = checkpoint
		b := bitShiftLeft(checkpoint)
		i.bitmap = []byte{b}
	} else {
		if checkpoint >= i.rangeFirstCheckpoint() && checkpoint <= i.rangeLastCheckpoint() {
			// Update the bit in existing range
			b := bitShiftLeft(checkpoint)
			loc := (checkpoint - i.rangeFirstCheckpoint()) / 8
			i.bitmap[loc] = i.bitmap[loc] | b

			if checkpoint < i.firstCheckpoint {
				i.firstCheckpoint = checkpoint
			}
			if checkpoint > i.lastCheckpoint {
				i.lastCheckpoint = checkpoint
			}
		} else {
			// Expand the bitmap
			if checkpoint < i.rangeFirstCheckpoint() {
				// ...to the left
				c := (i.rangeFirstCheckpoint() - checkpoint) / 8
				if (i.rangeFirstCheckpoint()-checkpoint)%8 != 0 {
					c++
				}
				newBytes := make([]byte, c)
				i.bitmap = append(newBytes, i.bitmap...)

				b := bitShiftLeft(checkpoint)
				i.bitmap[0] = i.bitmap[0] | b

				i.firstCheckpoint = checkpoint
			} else if checkpoint > i.rangeLastCheckpoint() {
				// ... to the right
				newBytes := make([]byte, (checkpoint-i.rangeLastCheckpoint())/8+1)
				i.bitmap = append(i.bitmap, newBytes...)
				b := bitShiftLeft(checkpoint)
				loc := (checkpoint - i.rangeFirstCheckpoint()) / 8
				i.bitmap[loc] = i.bitmap[loc] | b

				i.lastCheckpoint = checkpoint
			}
		}
	}

	return nil
}

//lint:ignore U1000 Ignore unused function temporarily
func (i *CheckpointIndex) isActive(checkpoint uint32) bool {
	if checkpoint >= i.firstCheckpoint && checkpoint <= i.lastCheckpoint {
		b := bitShiftLeft(checkpoint)
		loc := (checkpoint - i.rangeFirstCheckpoint()) / 8
		return i.bitmap[loc]&b != 0
	} else {
		return false
	}
}

func (i *CheckpointIndex) iterate(f func(checkpoint uint32)) error {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	if i.firstCheckpoint == 0 {
		return nil
	}

	f(i.firstCheckpoint)
	curr := i.firstCheckpoint

	for {
		var err error
		curr, err = i.nextActive(curr + 1)
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

func (i *CheckpointIndex) Merge(other *CheckpointIndex) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	var err error

	other.iterate(func(checkpoint uint32) {
		if err != nil {
			return
		}
		err = i.setActive(checkpoint)
	})

	return err
}

// NextActive returns the next checkpoint (inclusive) where this index is active.
func (i *CheckpointIndex) NextActive(checkpoint uint32) (uint32, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.nextActive(checkpoint)
}

func (i *CheckpointIndex) nextActive(checkpoint uint32) (uint32, error) {
	if i.firstCheckpoint == 0 || checkpoint > i.lastCheckpoint {
		// We're past the end.
		// TODO: Should this be an error? or how should we signal NONE here?
		return 0, io.EOF
	}

	if checkpoint < i.firstCheckpoint {
		checkpoint = i.firstCheckpoint
	}

	// Must be within the range, find the first non-zero after our start
	loc := (checkpoint - i.rangeFirstCheckpoint()) / 8

	// Is it in the same byte?
	if shift, ok := maxBitAfter(i.bitmap[loc], (checkpoint-1)%8); ok {
		return i.rangeFirstCheckpoint() + (loc * 8) + shift, nil
	}

	// Scan bytes after
	loc++
	for ; loc < uint32(len(i.bitmap)); loc++ {
		// Find the offset of the set bit
		if shift, ok := maxBitAfter(i.bitmap[loc], 0); ok {
			return i.rangeFirstCheckpoint() + (loc * 8) + shift, nil
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

func (i *CheckpointIndex) Buffer() *bytes.Buffer {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	xdrCheckpoint := xdr.CheckpointIndex{
		FirstCheckpoint: xdr.Uint32(i.firstCheckpoint),
		LastCheckpoint:  xdr.Uint32(i.lastCheckpoint),
		Bitmap:          i.bitmap,
	}

	b, err := xdrCheckpoint.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(b)
}

// Flush flushes the index data to byte slice in index format.
func (i *CheckpointIndex) Flush() []byte {
	return i.Buffer().Bytes()
}
