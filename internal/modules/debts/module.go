// Package debts — módulo de gestión de deudas y pagos para el agente Aria.
//
// Implementa las interfaces agent.DataReader y agent.DataWriter (Gen2).
// Single-user: el deudor se identifica por Name, no por UserID.
package debts

import (
	"context"
	"fmt"
	"strings"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// Module gestiona deudas y sus pagos vía herramientas del agente.
type Module struct {
	debts    domain.DebtRepository
	payments domain.DebtPaymentRepository
	db       agent.PortDB
	logger   *charm.Logger
}

// New crea un nuevo módulo de deudas.
func New() *Module { return &Module{} }

func (m *Module) Name() string     { return "debts" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.debts = deps.Debts
	m.payments = deps.DebtPayments
	m.db = deps.DB
	m.logger = deps.Logger
	return nil
}

// PromptSection inyecta las deudas pendientes en el system prompt del agente.
// Usa una sola query con GROUP BY para calcular pagos — sin N+1.
// Devuelve string vacío si no hay deudas pendientes.
func (m *Module) PromptSection(ctx context.Context, _ string) string {
	rows, err := m.db.QueryContext(ctx, `
		SELECT
			d.id,
			d.name,
			d.total_amount,
			d.state,
			COALESCE(SUM(p.amount_int), 0) AS total_paid
		FROM debts d
		LEFT JOIN debts_payments p ON p.debt_id = d.id
		WHERE d.state IN ('PENDING', 'PARTIAL')
		GROUP BY d.id
		ORDER BY d.id
	`)
	if err != nil {
		return ""
	}
	defer rows.Close()

	type pendingDebt struct {
		id         int64
		name       string
		totalCents int64
		state      string
		paidCents  int64
	}

	var pending []pendingDebt
	for rows.Next() {
		var pd pendingDebt
		if err := rows.Scan(&pd.id, &pd.name, &pd.totalCents, &pd.state, &pd.paidCents); err != nil {
			continue
		}
		pending = append(pending, pd)
	}
	if len(pending) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## DEUDAS PENDIENTES\n")
	for _, pd := range pending {
		total := fromCents(pd.totalCents)
		paid := fromCents(pd.paidCents)
		remaining := total.Sub(paid)
		fmt.Fprintf(&b, "- Deuda #%d (%s): total $%s, pagado $%s, saldo $%s — %s\n",
			pd.id, pd.name,
			total.StringFixed(0),
			paid.StringFixed(0),
			remaining.StringFixed(0),
			pd.state,
		)
	}
	return b.String()
}
