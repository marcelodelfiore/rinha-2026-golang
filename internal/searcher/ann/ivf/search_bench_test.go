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

func BenchmarkIVFSelectProbeCentroids1024x16(b *testing.B) {
	ds := loadBenchmarkDataset(b)

	idx, err := New(ds, Config{
		Clusters: 1024,
		Probes:   16,
	})
	if err != nil {
		b.Fatalf("build ivf index: %v", err)
	}

	query := benchmarkQuery(ds)

	var candidates [maxClusters]centroidCandidate

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx.selectProbeCentroids(query, &candidates)
	}
}

func TestSelectProbeCentroidsReturnsExpectedCount(t *testing.T) {
	ds := loadBenchmarkDataset(t)

	idx, err := New(ds, Config{
		Clusters: 1024,
		Probes:   16,
	})
	if err != nil {
		t.Fatalf("build ivf index: %v", err)
	}

	query := benchmarkQuery(ds)

	var candidates [maxClusters]centroidCandidate
	count := idx.selectProbeCentroids(query, &candidates)

	if count != 16 {
		t.Fatalf("expected 16 probe candidates, got %d", count)
	}
}

func TestSelectProbeCentroidsKeepsBestCandidates(t *testing.T) {
	ds := loadBenchmarkDataset(t)

	idx, err := New(ds, Config{
		Clusters: 1024,
		Probes:   16,
	})
	if err != nil {
		t.Fatalf("build ivf index: %v", err)
	}

	query := benchmarkQuery(ds)

	var candidates [maxClusters]centroidCandidate
	count := idx.selectProbeCentroids(query, &candidates)

	if count != idx.probes {
		t.Fatalf("expected %d candidates, got %d", idx.probes, count)
	}

	worstSelected := candidates[0].distance
	for i := 1; i < count; i++ {
		if candidates[i].distance > worstSelected {
			worstSelected = candidates[i].distance
		}
	}

	for c := 0; c < idx.clusters; c++ {
		offset := c * dataset.VectorDimensions

		distance := squaredEuclideanVector(
			query,
			idx.centroids[offset:offset+dataset.VectorDimensions],
		)

		if containsCentroidCandidate(candidates[:count], c) {
			continue
		}

		if distance < worstSelected {
			t.Fatalf(
				"centroid %d with distance %f should have been selected; worst selected distance is %f",
				c,
				distance,
				worstSelected,
			)
		}
	}
}

func containsCentroidCandidate(candidates []centroidCandidate, index int) bool {
	for _, candidate := range candidates {
		if candidate.index == index {
			return true
		}
	}

	return false
}

type testingHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}

func loadBenchmarkDataset(t testingHelper) *dataset.Dataset {
	t.Helper()

	path := os.Getenv("REFERENCES_PATH")
	if path == "" {
		path = "../../../../resources/references.json.gz"
	}

	ds, err := dataset.LoadReferences(path)
	if err != nil {
		t.Fatalf("load references: %v", err)
	}

	return ds
}

func benchmarkQuery(ds *dataset.Dataset) vector.Vector {
	var query vector.Vector

	// Use one real reference vector as a stable benchmark query.
	copy(query[:], ds.Vectors[0:dataset.VectorDimensions])

	return query
}
