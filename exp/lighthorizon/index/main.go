package index

import (
	"bytes"
	"strconv"
	"sync"
)

const CheckpointIndexVersion = 1

type CheckpointIndex struct {
	mutex           sync.Mutex
	bitmap          []byte
	firstCheckpoint uint32
	shift           uint32
}

func (i *CheckpointIndex) SetActive(checkpoint uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.firstCheckpoint == 0 {
		i.firstCheckpoint = checkpoint
		b := byte(1) << (8 - checkpoint%8)
		if checkpoint%8 == 0 {
			i.shift = 7
		} else {
			i.shift = checkpoint%8 - 1
		}
		i.bitmap = []byte{b}
	} else {
		lastCheckpoint := i.firstCheckpoint + uint32(len(i.bitmap))*8 - i.shift - 1

		if checkpoint >= i.firstCheckpoint && checkpoint <= lastCheckpoint {
			// Update the bit in existing range
			b := byte(1) << (8 - checkpoint%8)
			loc := (checkpoint - i.firstCheckpoint) / 8
			i.bitmap[loc] = i.bitmap[loc] | b
		} else {
			// Expand the map
			if checkpoint < i.firstCheckpoint {
				// Check if moving the shift left will be enough
				if i.firstCheckpoint-checkpoint <= i.shift {
					b := byte(1) << (8 - checkpoint%8)
					i.bitmap[0] = i.bitmap[0] | b
					i.shift = checkpoint%8 - 1
				} else {
					c := (i.firstCheckpoint - checkpoint - i.shift) / 8
					if (i.firstCheckpoint-checkpoint-i.shift)%8 > 0 {
						c++
					}
					newBytes := make([]byte, c)
					i.bitmap = append(newBytes, i.bitmap...)

					b := byte(1) << (8 - checkpoint%8)
					i.bitmap[0] = i.bitmap[0] | b
					if checkpoint%8 == 0 {
						i.shift = 7
					} else {
						i.shift = checkpoint%8 - 1
					}
				}
				i.firstCheckpoint = checkpoint
			} else if checkpoint > lastCheckpoint {
				newBytes := make([]byte, (checkpoint-lastCheckpoint)/8+1)
				i.bitmap = append(i.bitmap, newBytes...)
				b := byte(1) << (8 - checkpoint%8)
				loc := (checkpoint - i.firstCheckpoint) / 8
				i.bitmap[loc] = i.bitmap[loc] | b
			}
		}
	}

	return nil
}

func (i *CheckpointIndex) Bytes() []byte {
	return i.bitmap
}

func (i *CheckpointIndex) Shift() uint32 {
	return i.shift
}

func (i *CheckpointIndex) IsActive(checkpoint uint32) (bool, error) {
	return false, nil
}

func (i *CheckpointIndex) Buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteString(strconv.FormatInt(int64(i.firstCheckpoint), 10))
	b.WriteByte(0)
	b.Write(i.bitmap)
	return &b
}

// Flush flushes the index data to byte slice in index format.
func (i *CheckpointIndex) Flush() []byte {
	return i.Buffer().Bytes()
}
