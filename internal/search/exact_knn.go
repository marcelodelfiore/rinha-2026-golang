package search

import (
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

const fixedK = 5

type ExactKNN struct {
	dataset *dataset.Dataset
}

func (s *ExactKNN) SearchInto(query vector.Vector, out *[fixedK]Neighbor) int {
	count := 0

	for i := 0; i < s.dataset.Count; i++ {
		offset := s.dataset.VectorOffset(i)

		distance := squaredEuclidean(
			query,
			s.dataset.Vectors[offset:offset+dataset.VectorDimensions],
		)

		candidate := Neighbor{
			Distance: distance,
			Fraud:    s.dataset.Labels[i],
		}

		insertFixedNeighbor(out, &count, candidate)
	}

	return count
}
func NewExactKNN(dataset *dataset.Dataset) *ExactKNN {
	return &ExactKNN{dataset: dataset}
}

func (s *ExactKNN) Search(query vector.Vector, k int) []Neighbor {
	var best [fixedK]Neighbor
	count := s.SearchInto(query, &best)

	result := make([]Neighbor, count)
	copy(result, best[:count])

	return result
}

func squaredEuclidean(query vector.Vector, reference []float32) float32 {

	_ = query[13]
	_ = reference[13]

	d0 := query[0] - reference[0]
	d1 := query[1] - reference[1]
	d2 := query[2] - reference[2]
	d3 := query[3] - reference[3]
	d4 := query[4] - reference[4]
	d5 := query[5] - reference[5]
	d6 := query[6] - reference[6]
	d7 := query[7] - reference[7]
	d8 := query[8] - reference[8]
	d9 := query[9] - reference[9]
	d10 := query[10] - reference[10]
	d11 := query[11] - reference[11]
	d12 := query[12] - reference[12]
	d13 := query[13] - reference[13]

	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}

func insertFixedNeighbor(best *[fixedK]Neighbor, count *int, candidate Neighbor) {
	if *count < fixedK {
		best[*count] = candidate
		*count++
		sortFixedNeighbors(best, *count)
		return
	}

	if candidate.Distance >= best[fixedK-1].Distance {
		return
	}

	best[fixedK-1] = candidate
	sortFixedNeighbors(best, fixedK)
}

func sortFixedNeighbors(best *[fixedK]Neighbor, count int) {
	for i := 1; i < count; i++ {
		current := best[i]
		j := i - 1

		for j >= 0 && best[j].Distance > current.Distance {
			best[j+1] = best[j]
			j--
		}

		best[j+1] = current
	}
}

func manhattan14(query vector.Vector, reference []float32) float32 {
	_ = query[13]
	_ = reference[13]

	var sum float32

	d0 := query[0] - reference[0]
	if d0 < 0 { d0 = -d0 }
	sum += d0

	d1 := query[1] - reference[1]
	if d1 < 0 { d1 = -d1 }
	sum += d1

	d2 := query[2] - reference[2]
	if d2 < 0 { d2 = -d2 }
	sum += d2

	// ... repeat until d13

	d13 := query[13] - reference[13]
	if d13 < 0 { d13 = -d13 }
	sum += d13

	return sum
}
