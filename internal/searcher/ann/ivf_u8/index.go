package ivf_u8

import (
	"errors"
	"log"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
)

type Index struct {
	vectors []uint8
	labels  []uint8

	count int
	dims  int

	clusters int
	probes   int

	centroids []uint8

	// CSR-like cluster storage:
	//
	// cluster c contains indices:
	//   indices[offsets[c] : offsets[c+1]]
	offsets []int
	indices []int
}

func New(ds *dataset.ReferenceDataset, config Config) (*Index, error) {
	if ds == nil {
		return nil, errors.New("dataset is nil")
	}

	if !ds.IsUint8() {
		return nil, errors.New("ivf_u8 requires uint8 dataset")
	}

	if ds.Count <= 0 {
		return nil, errors.New("dataset count must be positive")
	}

	if ds.Dims <= 0 {
		return nil, errors.New("dataset dims must be positive")
	}

	expectedVectorLen := ds.Count * ds.Dims
	if len(ds.VectorsU8) != expectedVectorLen {
		return nil, errors.New("invalid uint8 vector length")
	}

	if len(ds.Labels) != ds.Count {
		return nil, errors.New("invalid labels length")
	}

	config = config.Normalize()

	if config.Clusters > ds.Count {
		config.Clusters = ds.Count
	}

	if config.Probes > config.Clusters {
		config.Probes = config.Clusters
	}

	idx := &Index{
		vectors:  ds.VectorsU8,
		labels:   ds.Labels,
		count:    ds.Count,
		dims:     ds.Dims,
		clusters: config.Clusters,
		probes:   config.Probes,
	}

	log.Printf(
		"building ivf_u8 index: count=%d dims=%d clusters=%d probes=%d",
		idx.count,
		idx.dims,
		idx.clusters,
		idx.probes,
	)

	idx.build()

	log.Printf(
		"built ivf_u8 index: clusters=%d indices=%d centroids=%d",
		idx.clusters,
		len(idx.indices),
		len(idx.centroids),
	)

	return idx, nil
}

func (idx *Index) build() {
	counts := make([]int, idx.clusters)
	sums := make([]uint64, idx.clusters*idx.dims)

	// First pass:
	// assign each vector to a deterministic bucket and accumulate centroid sums.
	for i := 0; i < idx.count; i++ {
		offset := i * idx.dims
		vector := idx.vectors[offset : offset+idx.dims]

		clusterID := hashVectorToCluster(vector, idx.clusters)
		counts[clusterID]++

		sumOffset := clusterID * idx.dims
		for j, value := range vector {
			sums[sumOffset+j] += uint64(value)
		}
	}

	idx.offsets = make([]int, idx.clusters+1)

	for c := 0; c < idx.clusters; c++ {
		idx.offsets[c+1] = idx.offsets[c] + counts[c]
	}

	idx.indices = make([]int, idx.count)

	writePositions := make([]int, idx.clusters)
	copy(writePositions, idx.offsets[:idx.clusters])

	// Second pass:
	// fill CSR index list.
	for i := 0; i < idx.count; i++ {
		offset := i * idx.dims
		vector := idx.vectors[offset : offset+idx.dims]

		clusterID := hashVectorToCluster(vector, idx.clusters)
		position := writePositions[clusterID]

		idx.indices[position] = i
		writePositions[clusterID]++
	}

	idx.centroids = make([]uint8, idx.clusters*idx.dims)

	for c := 0; c < idx.clusters; c++ {
		clusterCount := counts[c]
		centroidOffset := c * idx.dims

		if clusterCount == 0 {
			// Empty clusters should be rare with hash assignment, but keep
			// a deterministic fallback centroid anyway.
			fallbackIndex := c % idx.count
			fallbackOffset := fallbackIndex * idx.dims

			copy(
				idx.centroids[centroidOffset:centroidOffset+idx.dims],
				idx.vectors[fallbackOffset:fallbackOffset+idx.dims],
			)

			continue
		}

		sumOffset := c * idx.dims

		for j := 0; j < idx.dims; j++ {
			// Rounded integer mean.
			value := (sums[sumOffset+j] + uint64(clusterCount/2)) / uint64(clusterCount)

			if value > 255 {
				value = 255
			}

			idx.centroids[centroidOffset+j] = uint8(value)
		}
	}
}

func hashVectorToCluster(vector []uint8, clusters int) int {
	var hash uint32 = 2166136261

	for _, value := range vector {
		hash ^= uint32(value)
		hash *= 16777619
	}

	return int(hash % uint32(clusters))
}
