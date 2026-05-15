package ivf_u8

import (
	"math"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
)

func (idx *Index) SearchInto(query search.VectorU8, out *[search.FixedK]search.Neighbor) int {
	if len(query) != idx.dims {
		return 0
	}

	for i := 0; i < search.FixedK; i++ {
		out[i] = search.Neighbor{
			Index:    -1,
			Distance: math.MaxInt,
			Fraud:    false,
		}
	}

	selectedClusters := make([]int, idx.probes)
	selectedDistances := make([]int, idx.probes)

	for i := 0; i < idx.probes; i++ {
		selectedClusters[i] = -1
		selectedDistances[i] = math.MaxInt
	}

	idx.selectNearestCentroids(query, selectedClusters, selectedDistances)

	found := 0
	worstPos := 0
	worstDistance := math.MaxInt

	for _, clusterID := range selectedClusters {
		if clusterID < 0 {
			continue
		}

		start := idx.offsets[clusterID]
		end := idx.offsets[clusterID+1]

		for pos := start; pos < end; pos++ {
			refIndex := idx.indices[pos]
			vectorOffset := refIndex * idx.dims
			reference := idx.vectors[vectorOffset : vectorOffset+idx.dims]

			distance := squaredEuclideanU8(query, reference)

			if found < search.FixedK {
				out[found] = search.Neighbor{
					Index:    refIndex,
					Distance: distance,
					Fraud:    idx.labels[refIndex] == 1,
				}

				found++

				worstPos, worstDistance = findWorst(out, found)
				continue
			}

			if distance >= worstDistance {
				continue
			}

			out[worstPos] = search.Neighbor{
				Index:    refIndex,
				Distance: distance,
				Fraud:    idx.labels[refIndex] == 1,
			}

			worstPos, worstDistance = findWorst(out, found)
		}
	}

	sortNeighborsByDistance(out, found)

	return found
}

func (idx *Index) selectNearestCentroids(
	query search.VectorU8,
	selectedClusters []int,
	selectedDistances []int,
) {
	worstPos := 0
	worstDistance := math.MaxInt

	for c := 0; c < idx.clusters; c++ {
		offset := c * idx.dims
		centroid := idx.centroids[offset : offset+idx.dims]

		distance := squaredEuclideanU8(query, centroid)

		if distance >= worstDistance {
			continue
		}

		selectedClusters[worstPos] = c
		selectedDistances[worstPos] = distance

		worstPos, worstDistance = findWorstSelected(selectedDistances)
	}
}

func squaredEuclideanU8(a, b []uint8) int {
	var sum int

	for i := 0; i < len(a); i++ {
		diff := int(a[i]) - int(b[i])
		sum += diff * diff
	}

	return sum
}

func findWorstSelected(distances []int) (int, int) {
	worstPos := 0
	worstDistance := distances[0]

	for i := 1; i < len(distances); i++ {
		if distances[i] > worstDistance {
			worstPos = i
			worstDistance = distances[i]
		}
	}

	return worstPos, worstDistance
}

func findWorst(
	neighbors *[search.FixedK]search.Neighbor,
	count int,
) (int, int) {
	if count <= 0 {
		return 0, math.MaxInt
	}

	worstPos := 0
	worstDistance := neighbors[0].Distance

	for i := 1; i < count; i++ {
		if neighbors[i].Distance > worstDistance {
			worstPos = i
			worstDistance = neighbors[i].Distance
		}
	}

	return worstPos, worstDistance
}

func sortNeighborsByDistance(
	neighbors *[search.FixedK]search.Neighbor,
	count int,
) {
	for i := 1; i < count; i++ {
		current := neighbors[i]
		j := i - 1

		for j >= 0 && neighbors[j].Distance > current.Distance {
			neighbors[j+1] = neighbors[j]
			j--
		}

		neighbors[j+1] = current
	}
}
