package ivf

import (
	"os"
	"testing"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

func BenchmarkIVFSearchInto1024x16(b *testing.B) {
	ds := loadBenchmarkDataset(b)

	idx, err := New(ds, Config{
		Clusters: 1024,
		Probes:   16,
	})
	if err != nil {
		b.Fatalf("build ivf index: %v", err)
	}

	query := benchmarkQuery(ds)

	var neighbors [search.FixedK]search.Neighbor

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx.SearchInto(query, &neighbors)
	}
}

func loadBenchmarkDataset(b *testing.B) *dataset.Dataset {
	b.Helper()

	path := os.Getenv("REFERENCES_PATH")
	if path == "" {
		path = "../../../../resources/references.json.gz"
	}

	ds, err := dataset.LoadReferences(path)
	if err != nil {
		b.Fatalf("load references: %v", err)
	}

	return ds
}

func benchmarkQuery(ds *dataset.Dataset) vector.Vector {
	var query vector.Vector

	// Use one real reference vector as a stable benchmark query.
	copy(query[:], ds.Vectors[0:dataset.VectorDimensions])

	return query
}
