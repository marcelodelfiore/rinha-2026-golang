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
	var sum float32

	for i := 0; i < dataset.VectorDimensions; i++ {
		diff := query[i] - reference[i]
		sum += diff * diff
	}

	return sum
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
