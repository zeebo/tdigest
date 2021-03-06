package tdigest

import (
	"fmt"
	"math"
	"sort"
)

type summary struct {
	means  []float64
	counts []uint32
	bitree fen
}

func newSummary(initialCapacity uint) *summary {
	s := &summary{
		means:  make([]float64, 0, initialCapacity),
		counts: make([]uint32, 0, initialCapacity),
		bitree: fen{buf: make([]uint32, initialCapacity)},
	}
	return s
}

func (s summary) Len() int {
	return len(s.means)
}

func (s *summary) Add(key float64, value uint32) error {
	if math.IsNaN(key) {
		return fmt.Errorf("Key must not be NaN")
	}

	if value == 0 {
		return fmt.Errorf("Count must be >0")
	}

	idx := s.FindInsertionIndex(key)

	s.means = append(s.means, 0)
	copy(s.means[idx+1:], s.means[idx:])

	s.counts = append(s.counts, 0)
	copy(s.counts[idx+1:], s.counts[idx:])

	// man i wish there was a better way to handle an insertion, but Add
	// appears to happen far less than UpdateAt, and when Add does happen,
	// the size of keycounts is small. There may be some optimization here
	// by doing deltas, though.
	for i := idx + 1; i < len(s.means); i++ {
		s.bitree.Set(i, s.counts[i])
	}

	s.means[idx] = key
	s.counts[idx] = value
	s.bitree.Set(idx, value)

	return nil
}

func (s summary) Floor(x float64) int {
	i, j := 0, len(s.means)
	for i < j {
		h := int(uint(i+j) >> 1)
		if s.means[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i - 1
}

// Always insert to the right
func (s summary) FindInsertionIndex(x float64) int {
	return sort.Search(len(s.means), func(i int) bool {
		return s.means[i] > x
	})
}

func (s summary) HeadSum(index int) (sum float64) {
	return float64(s.bitree.Sum(index))
}

func (s summary) FindIndex(x float64) int {
	idx := sort.Search(len(s.means), func(i int) bool {
		return s.means[i] >= x
	})
	if idx < s.Len() && s.means[idx] == x {
		return idx
	}
	return s.Len()
}

func (s summary) Mean(uncheckedIndex int) float64 {
	return s.means[uncheckedIndex]
}

func (s summary) Count(uncheckedIndex int) uint32 {
	return s.counts[uncheckedIndex]
}

// return the index of the last item which the sum of counts
// of items before it is less than or equal to `sum`. -1 in
// case no centroid satisfies the requirement.
// Since it's cheap, this also returns the `HeadSum` until
// the found index (i.e. cumSum = HeadSum(FloorSum(x)))
func (s summary) FloorSum(sum float64) (index int, cumSum float64) {
	index = -1
	for i := 0; i < s.Len(); i++ {
		if cumSum <= sum {
			index = i
		} else {
			break
		}
		cumSum += float64(s.counts[i])
	}
	if index != -1 {
		cumSum -= float64(s.counts[index])
	}
	return index, cumSum
}

func (s *summary) setAt(index int, mean float64, count uint32) {
	s.means[index] = mean
	s.counts[index] = count
	s.adjustRight(index)
	s.adjustLeft(index)
	s.bitree.Set(index, count)
}

func (s *summary) adjustRight(index int) {
	for i := index + 1; i < len(s.means) && s.means[i-1] > s.means[i]; i++ {
		s.means[i-1], s.means[i] = s.means[i], s.means[i-1]
		s.counts[i-1], s.counts[i] = s.counts[i], s.counts[i-1]
	}
}

func (s *summary) adjustLeft(index int) {
	for i := index - 1; i >= 0 && s.means[i] > s.means[i+1]; i-- {
		s.means[i], s.means[i+1] = s.means[i+1], s.means[i]
		s.counts[i], s.counts[i+1] = s.counts[i+1], s.counts[i]
	}
}

func (s summary) ForEach(f func(float64, uint32) bool) {
	for i := 0; i < len(s.means); i++ {
		if !f(s.means[i], s.counts[i]) {
			break
		}
	}
}

func (s summary) Clone() *summary {
	return &summary{
		means:  append([]float64{}, s.means...),
		counts: append([]uint32{}, s.counts...),
		bitree: s.bitree.Clone(),
	}
}
