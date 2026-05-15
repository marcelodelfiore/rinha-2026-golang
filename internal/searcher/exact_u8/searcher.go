package exact_u8

import (
	"errors"
	"math"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
)

type Searcher struct {
	vectors []uint8
	labels  []uint8
	count   int
	dims    int
}

func New(ds *dataset.ReferenceDataset) (*Searcher, error) {
	if ds == nil {
		return nil, errors.New("dataset is nil")
	}

	if !ds.IsUint8() {
		return nil, errors.New("exact_u8 searcher requires uint8 dataset")
	}

	if ds.Count <= 0 {
		return nil, errors.New("dataset count must be positive")
	}

	if ds.Dims <= 0 {
		return nil, errors.New("dataset dims must be positive")
	}

	expectedVectorLen := ds.Count * ds.Dims
	if len(ds.VectorsU8) != expectedVectorLen {
		return nil, errors.New("invalid uint8 vector length")
	}

	if len(ds.Labels) != ds.Count {
		return nil, errors.New("invalid labels length")
	}

	return &Searcher{
		vectors: ds.VectorsU8,
		labels:  ds.Labels,
		count:   ds.Count,
		dims:    ds.Dims,
	}, nil
}

func (s *Searcher) SearchInto(query search.VectorU8, out *[search.FixedK]search.Neighbor) int {
	if len(query) != s.dims {
		return 0
	}

	for i := 0; i < search.FixedK; i++ {
		out[i] = search.Neighbor{
			Index:    -1,
			Distance: math.MaxInt,
			Fraud:    false,
		}
	}

	worstPos := 0
	worstDistance := math.MaxInt

	for i := 0; i < s.count; i++ {
		offset := i * s.dims
		reference := s.vectors[offset : offset+s.dims]

		distance := squaredEuclideanU8(query, reference)

		if distance >= worstDistance {
			continue
		}

		out[worstPos] = search.Neighbor{
			Index:    i,
			Distance: distance,
			Fraud:    s.labels[i] == 1,
		}

		worstPos, worstDistance = findWorst(out)
	}

	sortNeighborsByDistance(out)

	return search.FixedK
}

func squaredEuclideanU8(a, b []uint8) int {
	var sum int

	for i := 0; i < len(a); i++ {
		diff := int(a[i]) - int(b[i])
		sum += diff * diff
	}

	return sum
}

func findWorst(neighbors *[search.FixedK]search.Neighbor) (int, int) {
	worstPos := 0
	worstDistance := neighbors[0].Distance

	for i := 1; i < search.FixedK; i++ {
		if neighbors[i].Distance > worstDistance {
			worstPos = i
			worstDistance = neighbors[i].Distance
		}
	}

	return worstPos, worstDistance
}

func sortNeighborsByDistance(neighbors *[search.FixedK]search.Neighbor) {
	for i := 1; i < search.FixedK; i++ {
		current := neighbors[i]
		j := i - 1

		for j >= 0 && neighbors[j].Distance > current.Distance {
			neighbors[j+1] = neighbors[j]
			j--
		}

		neighbors[j+1] = current
	}
}
