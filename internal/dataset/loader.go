package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
)

type referenceRecord struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

type NormalizationConfig struct {
	MaxAmount            float32 `json:"max_amount"`
	MaxInstallments      float32 `json:"max_installments"`
	AmountVsAvgRatio     float32 `json:"amount_vs_avg_ratio"`
	MaxMinutes           float32 `json:"max_minutes"`
	MaxKm                float32 `json:"max_km"`
	MaxTxCount24h        float32 `json:"max_tx_count_24h"`
	MaxMerchantAvgAmount float32 `json:"max_merchant_avg_amount"`
}

func LoadReferences(path string) (*Dataset, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open references file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	var records []referenceRecord
	if err := json.NewDecoder(gzipReader).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode references json: %w", err)
	}

	ds := NewDataset(len(records))

	for i, record := range records {
		if len(record.Vector) != VectorDimensions {
			return nil, fmt.Errorf("record %d has %d dimensions, expected %d", i, len(record.Vector), VectorDimensions)
		}

		offset := ds.VectorOffset(i)
		copy(ds.Vectors[offset:offset+VectorDimensions], record.Vector)

		switch record.Label {
		case "fraud":
			ds.Labels[i] = true
		case "legit":
			ds.Labels[i] = false
		default:
			return nil, fmt.Errorf("record %d has invalid label %q", i, record.Label)
		}
	}

	return ds, nil
}

func LoadNormalization(path string) (NormalizationConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return NormalizationConfig{}, fmt.Errorf("open normalization file: %w", err)
	}
	defer file.Close()

	var config NormalizationConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return NormalizationConfig{}, fmt.Errorf("decode normalization json: %w", err)
	}

	return config, nil
}

func LoadMCCRisk(path string) (map[string]float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mcc_risk file: %w", err)
	}
	defer file.Close()

	var risk map[string]float32
	if err := json.NewDecoder(file).Decode(&risk); err != nil {
		return nil, fmt.Errorf("decode mcc_risk json: %w", err)
	}

	return risk, nil
}
