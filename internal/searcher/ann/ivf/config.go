package ivf

type Config struct {
	Clusters int
	Probes   int
}

func DefaultConfig() Config {
	return Config{
		Clusters: 128,
		Probes:   8,
	}
}
