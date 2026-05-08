package ivf

import (
	"math"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

const maxClusters = 2048

type centroidCandidate struct {
	index    int
	distance float32
}

func (idx *Index) SearchInto(query vector.Vector, out *[search.FixedK]search.Neighbor) int {
	var probeCandidates [maxClusters]centroidCandidate

	if idx.clusters > len(probeCandidates) {
		return 0
	}

	for i := 0; i < search.FixedK; i++ {
		out[i] = search.Neighbor{
			Distance: float32(math.Inf(1)),
			Index:    -1,
		}
	}

	probeCount := idx.selectProbeCentroids(query, &probeCandidates)

	count := 0

	for p := 0; p < probeCount; p++ {
		clusterIndex := probeCandidates[p].index
		cluster := idx.clusterData[clusterIndex]

		labelIndex := 0

		for offset := 0; offset < len(cluster.Vectors); offset += dataset.VectorDimensions {
			distance := squaredEuclideanVector(
				query,
				cluster.Vectors[offset:offset+dataset.VectorDimensions],
			)

			candidate := search.Neighbor{
				Distance: distance,
				Fraud:    cluster.Labels[labelIndex],
				Index:    -1,
			}

			insertFixedNeighbor(out, &count, candidate)

			labelIndex++
		}
	}
	return count
}

func (idx *Index) selectProbeCentroids(
	query vector.Vector,
	out *[maxClusters]centroidCandidate,
) int {
	count := 0
	worstIndex := 0

	for c := 0; c < idx.clusters; c++ {
		offset := c * dataset.VectorDimensions

		distance := squaredEuclideanVector(
			query,
			idx.centroids[offset:offset+dataset.VectorDimensions],
		)

		candidate := centroidCandidate{
			index:    c,
			distance: distance,
		}

		if count < idx.probes {
			out[count] = candidate

			if count == 0 || candidate.distance > out[worstIndex].distance {
				worstIndex = count
			}

			count++
			continue
		}

		if candidate.distance >= out[worstIndex].distance {
			continue
		}

		out[worstIndex] = candidate
		worstIndex = worstCentroidCandidateIndex(out, count)
	}

	return count
}

func (idx *Index) nearestCentroid(v []float32) int {
	bestIndex := 0
	bestDistance := float32(math.Inf(1))

	for c := 0; c < idx.clusters; c++ {
		offset := c * dataset.VectorDimensions

		distance := squaredEuclidean(
			v,
			idx.centroids[offset:offset+dataset.VectorDimensions],
		)

		if distance < bestDistance {
			bestDistance = distance
			bestIndex = c
		}
	}

	return bestIndex
}

func insertCentroidCandidate(
	out *[maxClusters]centroidCandidate,
	count *int,
	limit int,
	candidate centroidCandidate,
) {
	if *count < limit {
		out[*count] = candidate
		*count++

		for i := *count - 1; i > 0; i-- {
			if out[i].distance >= out[i-1].distance {
				break
			}

			out[i], out[i-1] = out[i-1], out[i]
		}

		return
	}

	if candidate.distance >= out[limit-1].distance {
		return
	}

	out[limit-1] = candidate

	for i := limit - 1; i > 0; i-- {
		if out[i].distance >= out[i-1].distance {
			break
		}

		out[i], out[i-1] = out[i-1], out[i]
	}
}

func insertFixedNeighbor(
	out *[search.FixedK]search.Neighbor,
	count *int,
	candidate search.Neighbor,
) {
	if *count < search.FixedK {
		out[*count] = candidate
		*count++

		for i := *count - 1; i > 0; i-- {
			if out[i].Distance >= out[i-1].Distance {
				break
			}

			out[i], out[i-1] = out[i-1], out[i]
		}

		return
	}

	if candidate.Distance >= out[search.FixedK-1].Distance {
		return
	}

	out[search.FixedK-1] = candidate

	for i := search.FixedK - 1; i > 0; i-- {
		if out[i].Distance >= out[i-1].Distance {
			break
		}

		out[i], out[i-1] = out[i-1], out[i]
	}
}

func squaredEuclidean(a, b []float32) float32 {
	var sum float32

	for i := 0; i < dataset.VectorDimensions; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

func squaredEuclideanVector(query vector.Vector, reference []float32) float32 {
	_ = query[13]
	_ = reference[13]

	d0 := query[0] - reference[0]
	d1 := query[1] - reference[1]
	d2 := query[2] - reference[2]
	d3 := query[3] - reference[3]
	d4 := query[4] - reference[4]
	d5 := query[5] - reference[5]
	d6 := query[6] - reference[6]
	d7 := query[7] - reference[7]
	d8 := query[8] - reference[8]
	d9 := query[9] - reference[9]
	d10 := query[10] - reference[10]
	d11 := query[11] - reference[11]
	d12 := query[12] - reference[12]
	d13 := query[13] - reference[13]

	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}

func worstCentroidCandidateIndex(
	out *[maxClusters]centroidCandidate,
	count int,
) int {
	worstIndex := 0
	worstDistance := out[0].distance

	for i := 1; i < count; i++ {
		if out[i].distance > worstDistance {
			worstDistance = out[i].distance
			worstIndex = i
		}
	}

	return worstIndex
}
