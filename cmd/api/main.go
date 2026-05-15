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
	"github.com/marcelodelfiore/rinha-2026-golang/internal/searcher/exact_u8"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vectorizer"
)

func main() {
	runtime.GOMAXPROCS(1)

	searchMode := envOrDefault("SEARCH_MODE", "exact_u8")

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

	var s search.Searcher

	switch searchMode {
	case "exact_u8":
		s, err = exact_u8.New(binaryDataset)
		if err != nil {
			log.Fatalf("build exact_u8 searcher: %v", err)
		}

	default:
		log.Fatalf("unsupported SEARCH_MODE for u8 runtime: %s", searchMode)
	}

	engine := fraud.NewEngine(v, s)

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
