// Package agent — messenger.go define la abstracción de entrega de mensajes.
// El Messenger desacopla al agente del canal concreto (WhatsApp, CLI, etc.).
package agent

import "context"

// Messenger define el contrato para la entrega de mensajes.
type Messenger interface {
	Send(ctx context.Context, msg Message) error
}

// Message representa un mensaje genérico enviable por el Messenger.
type Message interface {
	Recipient() string
	Type() string
	Payload() any
}

const (
	TypeChat  = "chat"
	TypeAudio = "audio"
)

// TextMessage implementa Message para texto plano.
type TextMessage struct {
	To      string
	Content string
}

func (m TextMessage) Recipient() string { return m.To }
func (m TextMessage) Type() string      { return TypeChat }
func (m TextMessage) Payload() any      { return m.Content }

// AudioMessage implementa Message para bytes de audio (OGG/Opus).
type AudioMessage struct {
	To   string
	Data []byte
}

func (m AudioMessage) Recipient() string { return m.To }
func (m AudioMessage) Type() string      { return TypeAudio }
func (m AudioMessage) Payload() any      { return m.Data }

var (
	_ Message = TextMessage{}
	_ Message = AudioMessage{}
)
