package vectorizer

import (
	"errors"
	"time"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/api"
	"github.com/marcelodelfiore/rinha-2026-golang/internal/vector"
)

var ErrInvalidInput = errors.New("invalid vectorizer input")

type MCCRiskTable map[string]float32

type Normalization struct {
	MaxAmount            float32
	MaxInstallments      float32
	AmountVsAvgRatio     float32
	MaxMinutes           float32
	MaxKm                float32
	MaxTxCount24h        float32
	MaxMerchantAvgAmount float32
}

type Vectorizer struct {
	normalization Normalization
	mccRisk       MCCRiskTable
}

func New(normalization Normalization, mccRisk MCCRiskTable) *Vectorizer {
	return &Vectorizer{
		normalization: normalization,
		mccRisk:       mccRisk,
	}
}

func (v *Vectorizer) Vectorize(input any) (vector.Vector, error) {
	request, ok := input.(api.FraudScoreRequest)
	if !ok {
		return vector.Vector{}, ErrInvalidInput
	}

	requestedAt, err := time.Parse(time.RFC3339, request.Transaction.RequestedAt)
	if err != nil {
		return vector.Vector{}, err
	}

	var result vector.Vector

	result[0] = normalize(float32(request.Transaction.Amount), v.normalization.MaxAmount)
	result[1] = normalize(float32(request.Transaction.Installments), v.normalization.MaxInstallments)

	if request.Customer.AvgAmount > 0 {
		amountVsAvg := float32(request.Transaction.Amount) / float32(request.Customer.AvgAmount)
		result[2] = normalize(amountVsAvg, v.normalization.AmountVsAvgRatio)
	}

	result[3] = clamp(float32(requestedAt.UTC().Hour()) / 23.0)
	result[4] = clamp(dayOfWeekMondayZero(requestedAt) / 6.0)

	if request.LastTransaction == nil {
		result[5] = 1
		result[6] = 1
	} else {
		lastTimestamp, err := time.Parse(time.RFC3339, request.LastTransaction.Timestamp)
		if err != nil {
			return vector.Vector{}, err
		}

		minutes := requestedAt.Sub(lastTimestamp).Minutes()
		result[5] = normalize(float32(minutes), v.normalization.MaxMinutes)
		result[6] = normalize(float32(request.LastTransaction.KmFromCurrent), v.normalization.MaxKm)
	}

	result[7] = normalize(float32(request.Terminal.KmFromHome), v.normalization.MaxKm)
	result[8] = normalize(float32(request.Customer.TxCount24h), v.normalization.MaxTxCount24h)

	if request.Terminal.IsOnline {
		result[9] = 1
	}

	if request.Terminal.CardPresent {
		result[10] = 1
	}

	if !knownMerchant(request.Merchant.ID, request.Customer.KnownMerchants) {
		result[11] = 1
	}

	result[12] = clamp(v.riskForMCC(request.Merchant.MCC))
	result[13] = normalize(float32(request.Merchant.AvgAmount), v.normalization.MaxMerchantAvgAmount)

	return result, nil
}

func normalize(value, max float32) float32 {
	if max <= 0 {
		return 0
	}

	return clamp(value / max)
}

func clamp(value float32) float32 {
	if value < 0 {
		return 0
	}

	if value > 1 {
		return 1
	}

	return value
}

func knownMerchant(merchantID string, knownMerchants []string) bool {
	for _, known := range knownMerchants {
		if known == merchantID {
			return true
		}
	}

	return false
}

func (v *Vectorizer) riskForMCC(mcc string) float32 {
	risk, ok := v.mccRisk[mcc]
	if !ok {
		return 0.5
	}

	return risk
}

func dayOfWeekMondayZero(t time.Time) float32 {
	weekday := t.UTC().Weekday()

	if weekday == time.Sunday {
		return 6
	}

	return float32(weekday - time.Monday)
}
