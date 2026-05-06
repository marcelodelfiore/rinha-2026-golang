package search

import "github.com/marcelodelfiore/rinha-2026-golang/internal/vector"

const FixedK = 5

type Searcher interface {
	SearchInto(query vector.Vector, out *[FixedK]Neighbor) int
}
