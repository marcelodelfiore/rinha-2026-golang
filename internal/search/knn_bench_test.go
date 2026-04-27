package search

import (
	"math/rand"
	"testing"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

var sinkBest [fixedK]Neighbor
var sinkCount int

func BenchmarkExactKNN_SearchInto_100k_14d(b *testing.B) {
	ds := &dataset.Dataset{
		Count:   100_000,
		Vectors: make([]float32, 100_000*dataset.VectorDimensions),
		Labels:  make([]bool, 100_000),
	}

	for i := range ds.Vectors {
		ds.Vectors[i] = rand.Float32()
	}

	for i := range ds.Labels {
		ds.Labels[i] = rand.Intn(2) == 1
	}

	knn := NewExactKNN(ds)

	var query vector.Vector
	for i := 0; i < dataset.VectorDimensions; i++ {
		query[i] = rand.Float32()
	}

	var out [fixedK]Neighbor

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sinkCount = knn.SearchInto(query, &out)
	}

	sinkBest = out
}
