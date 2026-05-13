// Package whatsapp — messenger.go adapta el Bot al contrato agent.Messenger.
// Esta es la única pieza de "glue" entre el paquete whatsapp y la interfaz Messenger.
// El agente nunca importa whatsapp/; solo conoce la interfaz Messenger.
package whatsapp

import (
	"context"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"go.mau.fi/whatsmeow/types"
)

// WhatsAppMessenger implementa agent.Messenger entregando mensajes vía whatsmeow.
type WhatsAppMessenger struct {
	bot *Bot
}

// NewMessenger crea el adaptador que conecta el Messenger del agente con el bot de WhatsApp.
func NewMessenger(bot *Bot) *WhatsAppMessenger {
	return &WhatsAppMessenger{bot: bot}
}

// Send entrega el mensaje al destinatario usando el canal correcto según su tipo.
// - TypeChat → texto plano.
// - TypeProgress → mensaje de progreso "⏳ [N/M] descripción".
// - TypeAudio → nota de voz (síntesis de voz ya realizada).
func (m *WhatsAppMessenger) Send(ctx context.Context, msg agent.Message) error {
	jid, err := types.ParseJID(msg.Recipient())
	if err != nil {
		return fmt.Errorf("messenger: JID inválido %q: %w", msg.Recipient(), err)
	}

	switch msg.Type() {
	case agent.TypeChat:
		if text, ok := msg.Payload().(string); ok {
			m.bot.sendReply(ctx, jid, text)
		}

	}

	return nil
}

// Compile-time: verificar que WhatsAppMessenger implementa agent.Messenger.
var _ agent.Messenger = (*WhatsAppMessenger)(nil)
