package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/config"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

type confusionMatrix struct {
	truePositive  int
	trueNegative  int
	falsePositive int
	falseNegative int
	total         int
}

func main() {
	skipSelf := flag.Bool("skip-self", false, "skip neighbor when neighbor index equals query index")
	limit := flag.Int("limit", 0, "maximum number of vectors to evaluate; 0 means all")
	flag.Parse()

	referencesPath := envOrDefault("REFERENCES_PATH", "resources/references.json.gz")
	rejectVotes := envInt("FRAUD_REJECT_VOTES", 3)

	ds, err := dataset.LoadReferences(referencesPath)
	if err != nil {
		log.Fatalf("load references: %v", err)
	}

	searcher, err := config.New(ds)
	if err != nil {
		log.Fatalf("build searcher: %v", err)
	}

	evaluationCount := ds.Count
	if *limit > 0 && *limit < evaluationCount {
		evaluationCount = *limit
	}

	startedAt := time.Now()

	matrix := evaluateDataset(ds, searcher, rejectVotes, *skipSelf, evaluationCount)

	elapsed := time.Since(startedAt)

	printResult(matrix, rejectVotes, *skipSelf, elapsed)
}

func evaluateDataset(
	ds *dataset.Dataset,
	searcher search.Searcher,
	rejectVotes int,
	skipSelf bool,
	evaluationCount int,
) confusionMatrix {
	var matrix confusionMatrix

	for i := 0; i < evaluationCount; i++ {
		query := queryVector(ds, i)

		var neighbors [search.FixedK]search.Neighbor
		count := searcher.SearchInto(query, &neighbors)

		fraudVotes := 0
		effectiveCount := 0

		for n := 0; n < count; n++ {
			if skipSelf && neighbors[n].Index == i {
				continue
			}

			effectiveCount++

			if neighbors[n].Fraud {
				fraudVotes++
			}
		}

		if effectiveCount == 0 {
			continue
		}

		actualFraud := ds.Labels[i]
		predictedFraud := fraudVotes >= rejectVotes

		matrix.total++

		switch {
		case actualFraud && predictedFraud:
			matrix.truePositive++
		case !actualFraud && !predictedFraud:
			matrix.trueNegative++
		case !actualFraud && predictedFraud:
			matrix.falsePositive++
		case actualFraud && !predictedFraud:
			matrix.falseNegative++
		}
	}

	return matrix
}

func queryVector(ds *dataset.Dataset, index int) vector.Vector {
	var query vector.Vector

	offset := ds.VectorOffset(index)
	copy(query[:], ds.Vectors[offset:offset+dataset.VectorDimensions])

	return query
}

func printResult(matrix confusionMatrix, rejectVotes int, skipSelf bool, elapsed time.Duration) {
	weightedErrors := matrix.falsePositive + 3*matrix.falseNegative

	failureCount := matrix.falsePositive + matrix.falseNegative
	failureRate := float64(failureCount) / float64(matrix.total)

	fmt.Printf("evaluation result\n")
	fmt.Printf("=================\n")
	fmt.Printf("fraud_reject_votes: %d\n", rejectVotes)
	fmt.Printf("skip_self: %t\n", skipSelf)
	fmt.Printf("elapsed: %s\n", elapsed)
	fmt.Printf("total: %d\n", matrix.total)
	fmt.Printf("\n")

	fmt.Printf("true_positive:  %d\n", matrix.truePositive)
	fmt.Printf("true_negative:  %d\n", matrix.trueNegative)
	fmt.Printf("false_positive: %d\n", matrix.falsePositive)
	fmt.Printf("false_negative: %d\n", matrix.falseNegative)
	fmt.Printf("\n")

	fmt.Printf("failure_count: %.0f\n", float64(failureCount))
	fmt.Printf("failure_rate: %.4f\n", failureRate)
	fmt.Printf("weighted_errors_E: %d\n", weightedErrors)
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
