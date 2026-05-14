package debts

import (
	"context"
	"fmt"
)

func (m *Module) Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "create_debt":
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("create_debt: name is required")
		}
		amountFloat, ok := args["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("create_debt: amount must be a number")
		}
		if amountFloat < 0 {
			return nil, fmt.Errorf("create_debt: amount cannot be negative (got %f)", amountFloat)
		}
		amountCents := int64(amountFloat * 100)
		description, _ := args["description"].(string)

		id, err := m.debts.Create(ctx, name, amountCents, description)
		if err != nil {
			return nil, fmt.Errorf("create_debt: %w", err)
		}
		m.logger.Info("Deuda creada", "id", id, "name", name, "amount_cents", amountCents)
		return map[string]any{"ok": true, "id": id, "name": name, "amount": amountFloat}, nil

	case "add_debt_payment":
		debtID, ok := args["debt_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("add_debt_payment: debt_id must be a number")
		}
		amountFloat, ok := args["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("add_debt_payment: amount must be a number")
		}
		amountCents := int64(amountFloat * 100)
		notes, _ := args["notes"].(string)
		paidAt, _ := args["paid_at"].(string)
		if paidAt == "" {
			paidAt = "now"
		}
		debtIDInt := int64(debtID)

		debt, err := m.debts.GetByID(ctx, debtIDInt)
		if err != nil {
			return nil, fmt.Errorf("add_debt_payment: %w", err)
		}
		totalPaid, err := m.payments.GetTotalPaid(ctx, debtIDInt)
		if err != nil {
			return nil, fmt.Errorf("add_debt_payment: %w", err)
		}
		pending := debt.TotalAmount.Sub(totalPaid)
		payment := fromCents(amountCents)
		if payment.GreaterThan(pending) {
			return nil, fmt.Errorf("add_debt_payment: el pago ($%s) supera el saldo pendiente ($%s)",
				payment.StringFixed(2), pending.StringFixed(2))
		}

		paymentID, err := m.payments.Create(ctx, debtIDInt, amountCents, notes, paidAt)
		if err != nil {
			return nil, fmt.Errorf("add_debt_payment: %w", err)
		}

		// Recalcular estado automáticamente después del pago.
		if d, err := m.debts.GetByID(ctx, debtIDInt); err == nil {
			if tp, err := m.payments.GetTotalPaid(ctx, debtIDInt); err == nil {
				var newState string
				switch {
				case tp.GreaterThanOrEqual(d.TotalAmount):
					newState = "PAID"
				case tp.IsPositive():
					newState = "PARTIAL"
				default:
					newState = "PENDING"
				}
				_ = m.debts.UpdateState(ctx, debtIDInt, newState)
			}
		}

		m.logger.Info("Pago registrado", "payment_id", paymentID, "debt_id", debtIDInt, "amount_cents", amountCents)
		return map[string]any{"ok": true, "id": paymentID, "debt_id": debtIDInt, "amount": amountFloat}, nil

	case "update_debt":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("update_debt: id must be a number")
		}
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("update_debt: name is required")
		}
		amountFloat, ok := args["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("update_debt: amount must be a number")
		}
		if amountFloat < 0 {
			return nil, fmt.Errorf("update_debt: amount cannot be negative (got %f)", amountFloat)
		}
		amountCents := int64(amountFloat * 100)
		description, _ := args["description"].(string)

		if err := m.debts.Update(ctx, int64(id), name, amountCents, description); err != nil {
			return nil, fmt.Errorf("update_debt: %w", err)
		}
		m.logger.Info("Deuda actualizada", "id", int64(id), "name", name)
		return map[string]any{"ok": true, "id": int64(id), "name": name, "amount": amountFloat}, nil

	case "update_debt_state":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("update_debt_state: id must be a number")
		}
		state, _ := args["state"].(string)
		switch state {
		case "PENDING", "PARTIAL", "PAID":
			// valid state
		default:
			return nil, fmt.Errorf("update_debt_state: estado inválido %q (debe ser PENDING, PARTIAL o PAID)", state)
		}
		if err := m.debts.UpdateState(ctx, int64(id), state); err != nil {
			return nil, fmt.Errorf("update_debt_state: %w", err)
		}
		return map[string]any{"ok": true, "id": int64(id), "state": state}, nil

	case "delete_debt":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("delete_debt: id must be a number")
		}
		if err := m.debts.Delete(ctx, int64(id)); err != nil {
			return nil, fmt.Errorf("delete_debt: %w", err)
		}
		m.logger.Info("Deuda eliminada", "id", int64(id))
		return map[string]any{"ok": true, "id": int64(id)}, nil

	case "update_debt_payment":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("update_debt_payment: id must be a number")
		}
		amountFloat, ok := args["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("update_debt_payment: amount must be a number")
		}
		amountCents := int64(amountFloat * 100)
		notes, _ := args["notes"].(string)
		paidAt, _ := args["paid_at"].(string)

		if err := m.payments.Update(ctx, int64(id), amountCents, notes, paidAt); err != nil {
			return nil, fmt.Errorf("update_debt_payment: %w", err)
		}
		m.logger.Info("Pago actualizado", "id", int64(id))
		return map[string]any{"ok": true, "id": int64(id), "amount": amountFloat}, nil

	case "delete_debt_payment":
		id, ok := args["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("delete_debt_payment: id must be a number")
		}
		paymentID := int64(id)

		// Necesitamos el debt_id para recalcular el estado después de borrar.
		payment, err := m.payments.GetByID(ctx, paymentID)
		if err != nil {
			return nil, fmt.Errorf("delete_debt_payment: %w", err)
		}
		debtID := payment.DebtID

		if err := m.payments.Delete(ctx, paymentID); err != nil {
			return nil, fmt.Errorf("delete_debt_payment: %w", err)
		}

		// Recalcular estado automáticamente después de eliminar el pago.
		if d, err := m.debts.GetByID(ctx, debtID); err == nil {
			if tp, err := m.payments.GetTotalPaid(ctx, debtID); err == nil {
				var newState string
				switch {
				case tp.GreaterThanOrEqual(d.TotalAmount):
					newState = "PAID"
				case tp.IsPositive():
					newState = "PARTIAL"
				default:
					newState = "PENDING"
				}
				_ = m.debts.UpdateState(ctx, debtID, newState)
			}
		}

		m.logger.Info("Pago eliminado", "id", paymentID, "debt_id", debtID)
		return map[string]any{"ok": true, "id": paymentID, "debt_id": debtID}, nil
	}
	return nil, fmt.Errorf("debts: herramienta de escritura desconocida %q", tool)
}
