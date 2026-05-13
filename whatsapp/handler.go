// Package whatsapp — handler.go maneja los eventos entrantes de whatsmeow.
package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleEvent es el callback registrado en whatsmeow.
func (b *Bot) handleEvent(evt interface{}) {
	msg, ok := evt.(*events.Message)
	if !ok {
		return
	}

	b.logger.Debug("Mensaje entrante", "sender", msg.Info.Sender, "chat", msg.Info.Chat)

	// Ignorar mensajes del pasado (offline queue al reconectar).
	if time.Since(msg.Info.Timestamp) > 30*time.Second {
		return
	}

	// Filtrado por modo (DM vs Grupo).
	if b.groupJID.IsEmpty() {
		if msg.Info.IsGroup {
			return
		}
	} else {
		if msg.Info.IsGroup && msg.Info.Chat != b.groupJID {
			return
		}
	}

	if msg.Info.IsFromMe {
		return
	}

	if !b.isAllowed(msg.Info.Sender) {
		b.logger.Warn("Usuario no autorizado", "sender", msg.Info.Sender)
		return
	}

	// Guardar el JID del admin automáticamente.
	senderJID := types.NewJID(msg.Info.Sender.User, msg.Info.Sender.Server).String()
	if err := b.configRepo.Set(context.Background(), "admin_jid", senderJID); err != nil {
		b.logger.Warn("No se pudo guardar admin_jid", "jid", senderJID, "err", err)
	} else {
		b.logger.Debug("Admin JID actualizado", "jid", senderJID)
	}

	go b.dispatchMessage(msg)
}

// dispatchMessage procesa un mensaje en background.
func (b *Bot) dispatchMessage(msg *events.Message) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Panic en handler", "sender", msg.Info.Sender, "panic", r)
			b.sendReply(ctx, msg.Info.Chat, "Ocurrió un error inesperado. Intentá de nuevo.")
		}
	}()

	_ = b.client.SendChatPresence(ctx, msg.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer b.client.SendChatPresence(ctx, msg.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText) //nolint:errcheck

	// Strip device part — "185092353872022:8@lid" → "185092353872022@lid".
	// WhatsApp rejects JIDs with a device component as message recipients.
	senderID := types.NewJID(msg.Info.Sender.User, msg.Info.Sender.Server).String()

	var text string

	// Audio → transcripción → responder como texto.
	if audioMsg := msg.Message.GetAudioMessage(); audioMsg != nil {
		data, err := b.client.Download(ctx, audioMsg)
		if err != nil {
			b.logger.Error("Error descargando audio", "err", err)
			b.sendReply(ctx, msg.Info.Chat, "No pude descargar tu nota de voz 🎙️❌")
			return
		}
		text, err = b.agent.TranscribeAudio(ctx, data)
		if err != nil {
			b.logger.Error("Error transcribiendo audio", "err", err)
			b.sendReply(ctx, msg.Info.Chat, "No pude transcribir tu nota de voz 🎙️❌")
			return
		}
		b.logger.Debug("Audio transcrito", "text_len", len(text))
	} else {
		text = getMessageText(msg)
	}

	if text == "" {
		return
	}

	b.processMessage(ctx, msg.Info.Chat, senderID, text)
}

// processMessage gestiona el ciclo de una conversación de texto.
func (b *Bot) processMessage(ctx context.Context, chat types.JID, senderID string, text string) {
	response, err := b.agent.Chat(ctx, senderID, text)
	if err != nil {
		b.logger.Error("Error en Chat", "sender", senderID, "err", err)
		b.sendReply(ctx, chat, "Error procesando tu consulta. Intentá de nuevo.")
		return
	}

	b.sendReply(ctx, chat, response)
}

func getMessageText(msg *events.Message) string {
	if c := msg.Message.GetConversation(); c != "" {
		return c
	}
	if ext := msg.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
