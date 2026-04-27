package detection

import "github.com/marcelodelfiore/rinha-2026-golang/internal/search"
import "github.com/marcelodelfiore/rinha-2026-golang/internal/vector"

const (
	DefaultK       = 5
	ApprovalCutoff = 0.6
)

type Vectorizer interface {
	Vectorize(input any) (vector.Vector, error)
}

type Engine struct {
	vectorizer Vectorizer
	searcher   search.Searcher
}

func NewEngine(vectorizer Vectorizer, searcher search.Searcher) *Engine {
	return &Engine{
		vectorizer: vectorizer,
		searcher:   searcher,
	}
}

func (e *Engine) Score(input any) (Result, error) {
	vector, err := e.vectorizer.Vectorize(input)
	if err != nil {
		return Result{}, err
	}

	neighbors := e.searcher.Search(vector, DefaultK)

	fraudCount := 0
	for _, neighbor := range neighbors {
		if neighbor.Fraud {
			fraudCount++
		}
	}

	fraudScore := float32(fraudCount) / float32(DefaultK)

	return Result{
		Approved:   fraudScore < ApprovalCutoff,
		FraudScore: fraudScore,
	}, nil
}
