package debts

import (
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
	"github.com/shopspring/decimal"
)

// fromCents convierte enteros (cents × 100) leídos de la DB a decimal.Decimal.
func fromCents(i int64) decimal.Decimal {
	return decimal.NewFromInt(i).Shift(-2)
}

// marshalDebt serializa una deuda para el LLM sin user_id.
func marshalDebt(d domain.Debt) map[string]any {
	return map[string]any{
		"id":           d.ID,
		"name":         d.Name,
		"total_amount": d.TotalAmount.StringFixed(0),
		"state":        d.State,
		"description":  d.Description,
	}
}

func marshalDebts(debts []domain.Debt) []map[string]any {
	out := make([]map[string]any, len(debts))
	for i, d := range debts {
		out[i] = marshalDebt(d)
	}
	return out
}
