package fraud

import (
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

const (
	DefaultK       = search.FixedK
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
	queryF32, err := e.vectorizer.Vectorize(input)
	if err != nil {
		return Result{}, err
	}

	var queryU8 [14]uint8
	vector.QuantizeToUint8(queryF32, queryU8[:])

	var neighbors [search.FixedK]search.Neighbor
	count := e.searcher.SearchInto(queryU8[:], &neighbors)

	if count == 0 {
		return Result{
			Approved:   true,
			FraudScore: 0,
		}, nil
	}

	fraudCount := 0
	for i := 0; i < count; i++ {
		if neighbors[i].Fraud {
			fraudCount++
		}
	}

	fraudScore := float32(fraudCount) / float32(count)

	return Result{
		Approved:   fraudScore < ApprovalCutoff,
		FraudScore: fraudScore,
	}, nil
}
