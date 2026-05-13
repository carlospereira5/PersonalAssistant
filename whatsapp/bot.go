// Package whatsapp integra whatsmeow para recibir y enviar mensajes de WhatsApp.
// bot.go contiene la definición de la estructura, construcción y ciclo de vida del bot.
package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strings"

	charm "github.com/charmbracelet/log"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// Bot es el wrapper de whatsmeow que conecta WhatsApp con Aria.
type Bot struct {
	client     *whatsmeow.Client
	agent      *agent.Aria
	allowed    map[string]bool // clave: número normalizado sin '+' ni servidor
	groupJID   types.JID
	configRepo domain.ConfigRepository
	logger     *charm.Logger
}

// New crea un Bot de WhatsApp listo para conectar.
// allowedNumbers son los números autorizados en formato "+5491112345678" o "5491112345678".
// groupJID es el JID del grupo donde Aria escucha (vacío = modo DM).
func New(ctx context.Context, ag *agent.Aria, dbPath string, allowedNumbers []string, groupJID string, logger *charm.Logger, cfgRepo domain.ConfigRepository) (*Bot, error) {
	// whatsmeow usa su propio logger; lo silenciamos excepto en WARN para no contaminar los logs.
	dbLog := waLog.Stdout("DB", "WARN", true)
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	container, err := sqlstore.New(ctx, "sqlite", dsn, dbLog)
	if err != nil {
		return nil, fmt.Errorf("whatsapp sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp device store: %w", err)
	}

	clientLog := waLog.Stdout("WA", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Normalizar números: quitar '+', espacios y sufijos de servidor (@s.whatsapp.net, etc.).
	allowed := make(map[string]bool, len(allowedNumbers))
	for _, num := range allowedNumbers {
		clean := strings.TrimSpace(num)
		clean = strings.TrimPrefix(clean, "+")
		if idx := strings.Index(clean, "@"); idx != -1 {
			clean = clean[:idx]
		}
		allowed[clean] = true
	}

	bot := &Bot{
		client:     client,
		agent:      ag,
		allowed:    allowed,
		configRepo: cfgRepo,
		logger:     logger.WithPrefix("WhatsApp"),
	}

	if groupJID != "" {
		parsed, err := types.ParseJID(groupJID)
		if err == nil && parsed.Server == types.GroupServer {
			bot.groupJID = parsed
			logger.Info("Modo grupo activo", "jid", parsed)
		} else {
			return nil, fmt.Errorf("WHATSAPP_GROUP_JID %q no es un grupo válido (@g.us)", groupJID)
		}
	} else {
		logger.Info("Modo DM activo (sin grupo configurado)")
	}

	return bot, nil
}

// SendTo envía un mensaje de texto a un número específico.
// phone es el número normalizado sin '+' ni servidor (ej: "5491112345678").
func (b *Bot) SendTo(ctx context.Context, phone, text string) {
	jid := types.NewJID(phone, types.DefaultUserServer)
	b.sendReply(ctx, jid, text)
}

// Broadcast envía un mensaje de texto a todos los números autorizados.
// Útil para alertas proactivas de stock enviadas desde agent/alerts.go.
func (b *Bot) Broadcast(ctx context.Context, text string) {
	b.logger.Info("Broadcast enviado", "destinatarios", len(b.allowed))
	for num := range b.allowed {
		jid := types.NewJID(num, types.DefaultUserServer)
		b.sendReply(ctx, jid, text)
	}
}

// Start conecta al servidor de WhatsApp y bloquea hasta que ctx se cancele.
// Si no hay sesión guardada, muestra un QR en la terminal para escanear.
func (b *Bot) Start(ctx context.Context) error {
	b.client.AddEventHandler(b.handleEvent)

	if b.client.Store.ID == nil {
		b.logger.Info("Sin sesión guardada — iniciando login por QR")
		if err := b.loginWithQR(ctx); err != nil {
			return fmt.Errorf("whatsapp QR login: %w", err)
		}
	} else {
		if err := b.client.Connect(); err != nil {
			return fmt.Errorf("whatsapp connect: %w", err)
		}
		b.logger.Info("Conectado con sesión existente")
	}

	<-ctx.Done()
	b.logger.Info("Desconectando...")
	b.client.Disconnect()
	return nil
}

// isAllowed verifica si el remitente está autorizado para usar Aria.
// Maneja también JIDs de tipo @lid (privacy-preserving IDs de WhatsApp),
// resolviendo el número real desde el store de whatsmeow.
func (b *Bot) isAllowed(jid types.JID) bool {
	if len(b.allowed) == 0 {
		return true
	}

	// Intento directo con el número normalizado.
	user := normalizeUser(jid.User)
	if b.allowed[user] {
		return true
	}

	// Si es un LID (@lid), intentar resolver el número de teléfono real.
	if jid.Server == types.HiddenUserServer {
		pn, err := b.client.Store.GetAltJID(context.Background(), jid)
		if err == nil && !pn.IsEmpty() {
			userPN := normalizeUser(pn.User)
			if b.allowed[userPN] {
				b.logger.Debug("LID resuelto a PN autorizado", "lid", jid.User, "pn", userPN)
				return true
			}
		}
	}

	return false
}

// normalizeUser limpia el User de un JID para comparación.
// Quita sufijos de multidispositivo como ".1", ":5", etc.
func normalizeUser(user string) string {
	if idx := strings.IndexAny(user, ".:"); idx != -1 {
		return user[:idx]
	}
	return user
}

// loginWithQR inicia el login mostrando un QR en la terminal.
// Bloquea hasta que el QR sea escaneado, expire o ctx se cancele.
func (b *Bot) loginWithQR(ctx context.Context) error {
	qrChan, _ := b.client.GetQRChannel(ctx)
	if err := b.client.Connect(); err != nil {
		return err
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			fmt.Print("\n  Escaneá este QR con WhatsApp:\n\n")
			qrterminal.GenerateWithConfig(evt.Code, qrterminal.Config{
				Level: qrterminal.L, Writer: os.Stdout, HalfBlocks: true,
				BlackChar: qrterminal.BLACK_BLACK, WhiteChar: qrterminal.WHITE_WHITE, QuietZone: 1,
			})
		case "success":
			b.logger.Info("Login QR exitoso")
			return nil
		case "timeout":
			// El QR expiró. El usuario debe reiniciar el bot.
			return fmt.Errorf("QR expirado — reiniciá el bot para generar uno nuevo")
		}
	}
	return nil
}
