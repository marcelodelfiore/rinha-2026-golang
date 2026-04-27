package search

import (
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

type ExactKNN struct {
	dataset *dataset.Dataset
}

func NewExactKNN(dataset *dataset.Dataset) *ExactKNN {
	return &ExactKNN{
		dataset: dataset,
	}
}

func (s *ExactKNN) Search(query vector.Vector, k int) []Neighbor {
	best := make([]Neighbor, 0, k)

	for i := 0; i < s.dataset.Count; i++ {
		offset := s.dataset.VectorOffset(i)
		distance := squaredEuclidean(query, s.dataset.Vectors[offset:offset+dataset.VectorDimensions])

		insertNeighbor(&best, Neighbor{
			Distance: distance,
			Fraud:    s.dataset.Labels[i],
		}, k)
	}

	return best
}

func squaredEuclidean(query vector.Vector, reference []float32) float32 {
	var sum float32

	for i := 0; i < dataset.VectorDimensions; i++ {
		diff := query[i] - reference[i]
		sum += diff * diff
	}

	return sum
}

func insertNeighbor(best *[]Neighbor, candidate Neighbor, k int) {
	neighbors := *best

	if len(neighbors) < k {
		neighbors = append(neighbors, candidate)
		*best = neighbors
		sortSmallNeighbors(*best)
		return
	}

	if candidate.Distance >= neighbors[k-1].Distance {
		return
	}

	neighbors[k-1] = candidate
	sortSmallNeighbors(neighbors)
}

func sortSmallNeighbors(neighbors []Neighbor) {
	for i := 1; i < len(neighbors); i++ {
		current := neighbors[i]
		j := i - 1

		for j >= 0 && neighbors[j].Distance > current.Distance {
			neighbors[j+1] = neighbors[j]
			j--
		}

		neighbors[j+1] = current
	}
}
