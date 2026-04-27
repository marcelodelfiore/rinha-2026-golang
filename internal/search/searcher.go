package search

import "github.com/marcelodelfiore/rinha-2026-golang/internal/vector"

type Neighbor struct {
	Distance float32
	Fraud    bool
}

type Searcher interface {
	Search(query vector.Vector, k int) []Neighbor
}
