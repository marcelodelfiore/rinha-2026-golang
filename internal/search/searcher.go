package search

const FixedK = 5

type VectorU8 []uint8

type Searcher interface {
	SearchInto(query VectorU8, out *[FixedK]Neighbor) int
}
