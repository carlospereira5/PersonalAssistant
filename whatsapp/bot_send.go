// Package whatsapp — bot_send.go contiene los métodos de envío de mensajes del Bot.
package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// sendReply envía un mensaje de texto plano al chat indicado.
func (b *Bot) sendReply(ctx context.Context, chat types.JID, text string) {
	_, err := b.client.SendMessage(ctx, chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		b.logger.Error("Error enviando mensaje", "chat", chat, "err", err)
	}
}
