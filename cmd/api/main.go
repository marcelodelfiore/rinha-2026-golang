package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/api"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/config"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/fraud"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/exact_u8"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vectorizer"
)

func main() {
	runtime.GOMAXPROCS(1)

	searchMode := envOrDefault("SEARCH_MODE", "ivf")

	referencesPath := envOrDefault("REFERENCES_PATH", "resources/references.json.gz")
	referencesBinPath := envOrDefault("REFERENCES_BIN_PATH", "resources/references_u8.bin")
	normalizationPath := envOrDefault("NORMALIZATION_PATH", "resources/normalization.json")
	mccRiskPath := envOrDefault("MCC_RISK_PATH", "resources/mcc_risk.json")

	normalizationConfig, err := dataset.LoadNormalization(normalizationPath)
	if err != nil {
		log.Fatalf("load normalization: %v", err)
	}

	mccRisk, err := dataset.LoadMCCRisk(mccRiskPath)
	if err != nil {
		log.Fatalf("load mcc risk: %v", err)
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

	var engine *fraud.Engine

	switch searchMode {
	case "exact_u8":
		log.Printf("loading uint8 binary references from %s", referencesBinPath)

		binaryDataset, err := dataset.LoadBinary(referencesBinPath)
		if err != nil {
			log.Fatalf("load binary references: %v", err)
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

		searcher, err := exact_u8.New(binaryDataset)
		if err != nil {
			log.Fatalf("build exact_u8 searcher: %v", err)
		}

		// IMPORTANT:
		// This only works if fraud.NewEngine accepts the same searcher interface
		// implemented by exact_u8.Searcher.
		engine = fraud.NewEngine(v, searcher)

	default:
		log.Printf("loading JSON references from %s using search mode %s", referencesPath, searchMode)

		referenceDataset, err := dataset.LoadReferences(referencesPath)
		if err != nil {
			log.Fatalf("load references: %v", err)
		}

		searcher, err := config.New(referenceDataset)
		if err != nil {
			log.Fatalf("build searcher: %v", err)
		}

		engine = fraud.NewEngine(v, searcher)
	}

	mux := http.NewServeMux()
	handler := api.NewHandler(engine)
	handler.RegisterRoutes(mux)

	port := envOrDefault("PORT", "8080")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Println("rinha api listening on :" + port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
