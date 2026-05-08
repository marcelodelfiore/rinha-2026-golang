package ivf

import (
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
)

type Cluster struct {
	Vectors []float32
	Labels  []bool
}

type Index struct {
	dataset *dataset.Dataset

	clusters int
	probes   int

	centroids []float32
	lists     [][]int

	clusterData []Cluster
}
