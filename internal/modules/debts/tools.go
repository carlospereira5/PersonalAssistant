package debts

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

// ReadTools devuelve las herramientas de lectura del módulo.
// Sin get_debts_by_user — simplificado para single-user.
func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "get_all_debts",
			Description: "Retorna todas las deudas con su estado y total pagado. Estado puede ser PENDING, PARTIAL o PAID.",
		},
		{
			Name:        "get_debt_by_id",
			Description: "Retorna una deuda específica con su detalle completo, incluyendo total pagado y saldo pendiente.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la deuda."},
			},
			Required: []string{"id"},
		},
		{
			Name:        "get_debt_payments",
			Description: "Retorna el historial de pagos de una deuda.",
			Parameters: []agentllm.ParamDef{
				{Name: "debt_id", Type: "integer", Description: "ID de la deuda."},
			},
			Required: []string{"debt_id"},
		},
	}
}

// WriteTools devuelve las herramientas de escritura del módulo.
func (m *Module) WriteTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "create_debt",
			Description: "Registra una nueva deuda. amount en pesos CLP (número entero, sin decimales).",
			Parameters: []agentllm.ParamDef{
				{Name: "name", Type: "string", Description: "Nombre del deudor (quien debe o a quien le deben)."},
				{Name: "amount", Type: "number", Description: "Monto total de la deuda en pesos CLP."},
				{Name: "description", Type: "string", Description: "Descripción opcional del concepto de la deuda."},
			},
			Required: []string{"name", "amount"},
		},
		{
			Name:        "add_debt_payment",
			Description: "Registra un pago parcial o total sobre una deuda. amount en pesos CLP. Si el pago completa la deuda, el estado se actualiza automáticamente a PAID.",
			Parameters: []agentllm.ParamDef{
				{Name: "debt_id", Type: "integer", Description: "ID de la deuda."},
				{Name: "amount", Type: "number", Description: "Monto pagado en pesos CLP."},
				{Name: "notes", Type: "string", Description: "Nota opcional sobre el pago."},
				{Name: "paid_at", Type: "string", Description: "Fecha del pago en formato YYYY-MM-DD. Si no se indica, se usa la fecha actual."},
			},
			Required: []string{"debt_id", "amount"},
		},
		{
			Name:        "update_debt",
			Description: "Actualiza los datos de una deuda existente: nombre, monto y/o descripción. El monto debe estar en pesos CLP (número entero). Usalo cuando el usuario quiera corregir o modificar una deuda.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la deuda a actualizar."},
				{Name: "name", Type: "string", Description: "Nuevo nombre del deudor."},
				{Name: "amount", Type: "number", Description: "Nuevo monto total en pesos CLP."},
				{Name: "description", Type: "string", Description: "Nueva descripción del concepto."},
			},
			Required: []string{"id", "name", "amount"},
		},
		{
			Name:        "update_debt_state",
			Description: "Actualiza el estado de una deuda. Estados válidos: PENDING, PARTIAL, PAID.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la deuda."},
				{Name: "state", Type: "string", Description: "Nuevo estado.", Enum: []string{"PENDING", "PARTIAL", "PAID"}},
			},
			Required: []string{"id", "state"},
		},
		{
			Name:        "delete_debt",
			Description: "Elimina una deuda y todos sus pagos registrados (borrado en cascada). Requiere confirmación del usuario antes de ejecutar.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la deuda a eliminar."},
			},
			Required: []string{"id"},
		},
		{
			Name:        "update_debt_payment",
			Description: "Actualiza los datos de un pago existente: monto, nota y/o fecha. El monto debe estar en pesos CLP (número entero). Usalo cuando el usuario haya ingresado mal un pago.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID del pago a actualizar."},
				{Name: "amount", Type: "number", Description: "Nuevo monto del pago en pesos CLP."},
				{Name: "notes", Type: "string", Description: "Nueva nota o descripción del pago."},
				{Name: "paid_at", Type: "string", Description: "Nueva fecha del pago en formato YYYY-MM-DD."},
			},
			Required: []string{"id", "amount"},
		},
		{
			Name:        "delete_debt_payment",
			Description: "Elimina un pago específico de una deuda. El estado de la deuda se recalcula automáticamente.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID del pago a eliminar."},
			},
			Required: []string{"id"},
		},
	}
}
