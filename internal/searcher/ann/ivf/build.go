package ivf

import (
	"fmt"
	"os"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
)

func New(ds *dataset.Dataset, cfg Config) (*Index, error) {
	if ds == nil {
		return nil, fmt.Errorf("ivf: nil dataset")
	}

	if cfg.Clusters <= 0 {
		return nil, fmt.Errorf("ivf: clusters must be > 0")
	}

	if cfg.Probes <= 0 {
		return nil, fmt.Errorf("ivf: probes must be > 0")
	}

	if cfg.Probes > cfg.Clusters {
		cfg.Probes = cfg.Clusters
	}

	idx := &Index{
		dataset:   ds,
		clusters:  cfg.Clusters,
		probes:    cfg.Probes,
		centroids: make([]float32, cfg.Clusters*dataset.VectorDimensions),
		lists:     make([][]int, cfg.Clusters),
	}

	idx.initializeCentroids()
	idx.assignVectors()
	idx.logClusterStats()

	return idx, nil
}

func (idx *Index) initializeCentroids() {
	step := idx.dataset.Count / idx.clusters
	if step <= 0 {
		step = 1
	}

	for c := 0; c < idx.clusters; c++ {
		srcIndex := c * step
		if srcIndex >= idx.dataset.Count {
			srcIndex = idx.dataset.Count - 1
		}

		srcOffset := idx.dataset.VectorOffset(srcIndex)
		dstOffset := c * dataset.VectorDimensions

		copy(
			idx.centroids[dstOffset:dstOffset+dataset.VectorDimensions],
			idx.dataset.Vectors[srcOffset:srcOffset+dataset.VectorDimensions],
		)
	}
}

func (idx *Index) assignVectors() {
	for i := 0; i < idx.dataset.Count; i++ {
		vectorOffset := idx.dataset.VectorOffset(i)

		cluster := idx.nearestCentroid(
			idx.dataset.Vectors[vectorOffset : vectorOffset+dataset.VectorDimensions],
		)

		idx.lists[cluster] = append(idx.lists[cluster], i)
	}
}

func (idx *Index) logClusterStats() {
	if os.Getenv("IVF_LOG_CLUSTER_STATS") != "1" {
		return
	}

	min := idx.dataset.Count
	max := 0
	total := 0
	empty := 0

	for _, list := range idx.lists {
		size := len(list)

		if size == 0 {
			empty++
		}

		if size < min {
			min = size
		}

		if size > max {
			max = size
		}

		total += size
	}

	avg := float64(total) / float64(idx.clusters)

	fmt.Printf(
		"ivf cluster stats: clusters=%d probes=%d min=%d max=%d avg=%.1f empty=%d\n",
		idx.clusters,
		idx.probes,
		min,
		max,
		avg,
		empty,
	)
}
