package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/ann/ivf"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"	
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/exact"
)

func New(ds *dataset.Dataset) (search.Searcher, error) {
	switch os.Getenv("SEARCH_MODE") {
	case "", "exact":
		return exact.NewExactKNN(ds), nil

	case "ivf":
		clusters := envInt("IVF_CLUSTERS", 128)
		probes := envInt("IVF_PROBES", 8)

		return ivf.New(ds, ivf.Config{
			Clusters: clusters,
			Probes:   probes,
		})

	default:
		return nil, fmt.Errorf("unknown SEARCH_MODE: %s", os.Getenv("SEARCH_MODE"))
	}
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
