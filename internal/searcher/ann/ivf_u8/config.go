package ivf_u8

import (
	"os"
	"strconv"
)

const (
	defaultClusters = 1024
	defaultProbes   = 16
)

type Config struct {
	Clusters int
	Probes   int
}

func DefaultConfig() Config {
	return Config{
		Clusters: defaultClusters,
		Probes:   defaultProbes,
	}
}

func ConfigFromEnv() Config {
	return Config{
		Clusters: envInt("IVF_CLUSTERS", defaultClusters),
		Probes:   envInt("IVF_PROBES", defaultProbes),
	}
}

func (c Config) Normalize() Config {
	if c.Clusters <= 0 {
		c.Clusters = defaultClusters
	}

	if c.Probes <= 0 {
		c.Probes = defaultProbes
	}

	if c.Probes > c.Clusters {
		c.Probes = c.Clusters
	}

	return c
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
