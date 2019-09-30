// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func (r Range) allCheckpoints() []uint32 {
	var s []uint32
	for chk := range r.Checkpoints() {
		s = append(s, chk)
	}
	return s
}

func TestRangeSize(t *testing.T) {
	assert.Equal(t, 1,
		MakeRange(0x3f, 0x3f).Size())

	assert.Equal(t, 2,
		MakeRange(0x3f, 0x7f).Size())

	assert.Equal(t, 2,
		MakeRange(0, 100).Size())

	assert.Equal(t, 4,
		MakeRange(0xff3f, 0xffff).Size())
}

func TestRangeEnumeration(t *testing.T) {
	assert.Equal(t,
		[]uint32{0x3f, 0x7f},
		MakeRange(0x3f, 0x7f).allCheckpoints())

	assert.Equal(t,
		[]uint32{0x3f},
		MakeRange(0x3f, 0x3f).allCheckpoints())

	assert.Equal(t,
		[]uint32{0x3f},
		MakeRange(0, 0).allCheckpoints())

	assert.Equal(t,
		[]uint32{0x3f, 0x7f},
		MakeRange(0, 0x40).allCheckpoints())

	assert.Equal(t,
		[]uint32{0xff},
		MakeRange(0xff, 0x40).allCheckpoints())
}

func TestFmtRangeList(t *testing.T) {

	assert.Equal(t,
		"",
		fmtRangeList([]uint32{}))

	assert.Equal(t,
		"0x0000003f",
		fmtRangeList([]uint32{0x3f}))

	assert.Equal(t,
		"[0x0000003f-0x0000007f]",
		fmtRangeList([]uint32{0x3f, 0x7f}))

	assert.Equal(t,
		"[0x0000003f-0x000000bf]",
		fmtRangeList([]uint32{0x3f, 0x7f, 0xbf}))

	assert.Equal(t,
		"[0x0000003f-0x0000007f], 0x000000ff",
		fmtRangeList([]uint32{0x3f, 0x7f, 0xff}))

	assert.Equal(t,
		"[0x0000003f-0x0000007f], [0x000000ff-0x0000017f]",
		fmtRangeList([]uint32{0x3f, 0x7f, 0xff, 0x13f, 0x17f}))

	assert.Equal(t,
		"[0x0000003f-0x0000007f], 0x000000ff, [0x0000017f-0x000001bf]",
		fmtRangeList([]uint32{0x3f, 0x7f, 0xff, 0x17f, 0x1bf}))
}
