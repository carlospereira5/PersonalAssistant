// Package agent — args.go exporta helpers para extraer argumentos de tool calls.
package agent

import (
	"context"
	"math"
)

// StrArg extrae un argumento string del mapa de args del LLM.
func StrArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

// FloatArg extrae un argumento numérico (JSON deserializa como float64).
func FloatArg(args map[string]any, key string) float64 {
	switch v := args[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

// IntArg extrae un entero con valor por defecto.
func IntArg(args map[string]any, key string, def int) int {
	v := FloatArg(args, key)
	if v == 0 {
		return def
	}
	return int(v)
}

// Int64Arg extrae un ID numérico con bounds check para evitar overflow.
func Int64Arg(args map[string]any, key string) int64 {
	v := FloatArg(args, key)
	if v <= 0 {
		return 0
	}
	if v >= math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// ctxKey es el tipo de clave para los valores de contexto de Aria.
type ctxKey struct{}

// UserIDFromCtx extrae el userID del contexto.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

func ctxWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}
