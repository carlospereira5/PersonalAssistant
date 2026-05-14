package debts

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "get_all_debts":
		debts, err := m.debts.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("get_all_debts: %w", err)
		}
		return map[string]any{"debts": marshalDebts(debts), "total": len(debts)}, nil

	case "get_debt_by_id":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("get_debt_by_id: id must be a number")
		}
		parsedID := int64(id)
		d, err := m.debts.GetByID(ctx, parsedID)
		if err != nil {
			return nil, fmt.Errorf("get_debt_by_id: %w", err)
		}
		result := marshalDebt(d)
		totalPaid, _ := m.payments.GetTotalPaid(ctx, parsedID)
		result["total_paid"] = totalPaid.StringFixed(0)
		result["remaining"] = d.TotalAmount.Sub(totalPaid).StringFixed(0)
		return result, nil

	case "get_debt_payments":
		debtID, ok := args["debt_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("get_debt_payments: debt_id must be a number")
		}
		pmts, err := m.payments.GetByDebt(ctx, int64(debtID))
		if err != nil {
			return nil, fmt.Errorf("get_debt_payments: %w", err)
		}
		list := make([]map[string]any, len(pmts))
		for i, p := range pmts {
			list[i] = map[string]any{
				"id":         p.ID,
				"debt_id":    p.DebtID,
				"amount":     p.Amount.StringFixed(0),
				"notes":      p.Notes,
				"paid_at":    p.PaidAt.Format("2006-01-02"),
				"created_at": p.CreatedAt.Format("2006-01-02"),
			}
		}
		return map[string]any{"payments": list, "total": len(list)}, nil
	}
	return nil, fmt.Errorf("debts: herramienta de lectura desconocida %q", tool)
}
