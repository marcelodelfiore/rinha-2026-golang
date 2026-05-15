package config

import (
	"errors"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
)

var ErrLegacySearchConfigDisabled = errors.New("legacy float32 search config is disabled; use u8 searchers")

func New(_ *dataset.Dataset) (search.Searcher, error) {
	return nil, ErrLegacySearchConfigDisabled
}
