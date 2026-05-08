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

func BenchmarkInsertFixedNeighbor(b *testing.B) {
	var neighbors [search.FixedK]search.Neighbor

	candidates := [...]search.Neighbor{
		{Distance: 0.90, Fraud: false, Index: 0},
		{Distance: 0.10, Fraud: true, Index: 1},
		{Distance: 0.70, Fraud: false, Index: 2},
		{Distance: 0.20, Fraud: true, Index: 3},
		{Distance: 0.50, Fraud: false, Index: 4},
		{Distance: 0.30, Fraud: true, Index: 5},
		{Distance: 0.80, Fraud: false, Index: 6},
		{Distance: 0.40, Fraud: true, Index: 7},
		{Distance: 0.60, Fraud: false, Index: 8},
		{Distance: 0.05, Fraud: true, Index: 9},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		count := 0

		for _, candidate := range candidates {
			insertFixedNeighbor(&neighbors, &count, candidate)
		}
	}
}

func TestInsertFixedNeighborKeepsBestFive(t *testing.T) {
	var neighbors [search.FixedK]search.Neighbor
	count := 0

	candidates := []search.Neighbor{
		{Distance: 0.90, Fraud: false, Index: 0},
		{Distance: 0.10, Fraud: true, Index: 1},
		{Distance: 0.70, Fraud: false, Index: 2},
		{Distance: 0.20, Fraud: true, Index: 3},
		{Distance: 0.50, Fraud: false, Index: 4},
		{Distance: 0.30, Fraud: true, Index: 5},
		{Distance: 0.80, Fraud: false, Index: 6},
		{Distance: 0.40, Fraud: true, Index: 7},
		{Distance: 0.60, Fraud: false, Index: 8},
		{Distance: 0.05, Fraud: true, Index: 9},
	}

	for _, candidate := range candidates {
		insertFixedNeighbor(&neighbors, &count, candidate)
	}

	if count != search.FixedK {
		t.Fatalf("expected count %d, got %d", search.FixedK, count)
	}

	maxSelectedDistance := neighbors[0].Distance
	for i := 1; i < count; i++ {
		if neighbors[i].Distance > maxSelectedDistance {
			maxSelectedDistance = neighbors[i].Distance
		}
	}

	if maxSelectedDistance != 0.40 {
		t.Fatalf("expected worst selected distance 0.40, got %f", maxSelectedDistance)
	}

	expectedIndexes := map[int]bool{
		1: true, // 0.10
		3: true, // 0.20
		5: true, // 0.30
		7: true, // 0.40
		9: true, // 0.05
	}

	for i := 0; i < count; i++ {
		if !expectedIndexes[neighbors[i].Index] {
			t.Fatalf("unexpected selected neighbor: index=%d distance=%f", neighbors[i].Index, neighbors[i].Distance)
		}
	}
}
