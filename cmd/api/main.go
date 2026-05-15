package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/api"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/fraud"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/ann/ivf_u8"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/exact_u8"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vectorizer"
)

func main() {
	runtime.GOMAXPROCS(1)

	mux := http.NewServeMux()
	handler := api.NewHandler(nil)
	handler.RegisterRoutes(mux)

	port := envOrDefault("PORT", "8080")

	go func() {
		log.Println("initializing fraud engine")

		engine, err := buildEngine()
		if err != nil {
			log.Fatalf("initialize fraud engine: %v", err)
		}

		handler.SetEngine(engine)

		log.Println("fraud engine ready")
	}()

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Println("rinha api listening on :" + port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func buildEngine() (*fraud.Engine, error) {
	searchMode := envOrDefault("SEARCH_MODE", "exact_u8")

	referencesBinPath := envOrDefault("REFERENCES_BIN_PATH", "resources/references_u8.bin")
	normalizationPath := envOrDefault("NORMALIZATION_PATH", "resources/normalization.json")
	mccRiskPath := envOrDefault("MCC_RISK_PATH", "resources/mcc_risk.json")

	normalizationConfig, err := dataset.LoadNormalization(normalizationPath)
	if err != nil {
		return nil, err
	}

	mccRisk, err := dataset.LoadMCCRisk(mccRiskPath)
	if err != nil {
		return nil, err
	}

	v := vectorizer.New(
		vectorizer.Normalization{
			MaxAmount:            normalizationConfig.MaxAmount,
			MaxInstallments:      normalizationConfig.MaxInstallments,
			AmountVsAvgRatio:     normalizationConfig.AmountVsAvgRatio,
			MaxMinutes:           normalizationConfig.MaxMinutes,
			MaxKm:                normalizationConfig.MaxKm,
			MaxTxCount24h:        normalizationConfig.MaxTxCount24h,
			MaxMerchantAvgAmount: normalizationConfig.MaxMerchantAvgAmount,
		},
		mccRisk,
	)

	log.Printf("loading uint8 binary references from %s", referencesBinPath)

	binaryDataset, err := dataset.LoadBinary(referencesBinPath)
	if err != nil {
		return nil, err
	}

	if !binaryDataset.IsUint8() {
		log.Fatalf("expected uint8 binary dataset, got format=%d", binaryDataset.Format)
	}

	log.Printf(
		"loaded uint8 dataset: count=%d dims=%d vectors=%d labels=%d",
		binaryDataset.Count,
		binaryDataset.Dims,
		len(binaryDataset.VectorsU8),
		len(binaryDataset.Labels),
	)

	var s search.Searcher

	switch searchMode {
	case "exact_u8":
		s, err = exact_u8.New(binaryDataset)
		if err != nil {
			return nil, err
		}

	case "ivf_u8":
		s, err = ivf_u8.New(binaryDataset, ivf_u8.ConfigFromEnv())
		if err != nil {
			return nil, err
		}

	default:
		log.Fatalf("unsupported SEARCH_MODE for u8 runtime: %s", searchMode)
	}

	return fraud.NewEngine(v, s), nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
