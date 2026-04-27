package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/api"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/detection"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/search"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vectorizer"
)

func main() {
	referencesPath := envOrDefault("REFERENCES_PATH", "resources/references.json.gz")
	normalizationPath := envOrDefault("NORMALIZATION_PATH", "resources/normalization.json")
	mccRiskPath := envOrDefault("MCC_RISK_PATH", "resources/mcc_risk.json")

	referenceDataset, err := dataset.LoadReferences(referencesPath)
	if err != nil {
		log.Fatalf("load references: %v", err)
	}

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

	searcher := search.NewExactKNN(referenceDataset)
	engine := detection.NewEngine(v, searcher)

	mux := http.NewServeMux()
	handler := api.NewHandler(engine)
	handler.RegisterRoutes(mux)

	port := envOrDefault("PORT", "8080")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Println("pprof listening on :6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
