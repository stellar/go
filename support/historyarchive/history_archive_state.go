// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

const NumLevels = 11

type HistoryArchiveState struct {
	Version        int    `json:"version"`
	Server         string `json:"server"`
	CurrentLedger  uint32 `json:"currentLedger"`
	CurrentBuckets [NumLevels]struct {
		Curr string `json:"curr"`
		Snap string `json:"snap"`
		Next struct {
			State  uint32 `json:"state"`
			Output string `json:"output,omitempty"`
		} `json:"next"`
	} `json:"currentBuckets"`
}

func (h *HistoryArchiveState) LevelSummary() (string, int, error) {
	summ := ""
	nz := 0
	for _, b := range h.CurrentBuckets {
		state := '_'
		for _, bs := range []string{
			b.Curr, b.Snap, b.Next.Output,
		} {
			// Ignore empty values
			if bs == "" {
				continue
			}

			h, err := DecodeHash(bs)
			if err != nil {
				return summ, nz, err
			}

			if !h.IsZero() {
				state = '#'
			}
		}
		if state != '_' {
			nz += 1
		}
		summ += string(state)
	}
	return summ, nz, nil
}

func (h *HistoryArchiveState) Buckets() ([]Hash, error) {
	r := []Hash{}
	for _, b := range h.CurrentBuckets {
		for _, bs := range []string{
			b.Curr, b.Snap, b.Next.Output,
		} {
			// Ignore empty values
			if bs == "" {
				continue
			}

			h, err := DecodeHash(bs)
			if err != nil {
				return r, err
			}
			if !h.IsZero() {
				r = append(r, h)
			}
		}
	}
	return r, nil
}

func (h *HistoryArchiveState) Range() Range {
	return Range{Low: 63, High: h.CurrentLedger}
}
