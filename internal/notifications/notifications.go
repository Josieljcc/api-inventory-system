package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type NotificationEvent struct {
	Type    string                 // e.g. "low_stock", "user_registered"
	To      string                 // email, phone, chat_id, etc
	Message string                 // main message
	Data    map[string]interface{} // extra payload
}

type NotificationSender interface {
	Send(event NotificationEvent) error
}

type NotificationService struct {
	senders []NotificationSender
}

func NewNotificationService(senders ...NotificationSender) *NotificationService {
	return &NotificationService{senders: senders}
}

func (ns *NotificationService) Notify(event NotificationEvent) {
	for _, sender := range ns.senders {
		_ = sender.Send(event) // ignore error for now, could log
	}
}

func NotifyLowStock(productName string, quantity int) {
	log.Printf("Atenção: Produto %s com estoque baixo (%d unidades)", productName, quantity)
}

type LogSender struct{}

func (l *LogSender) Send(event NotificationEvent) error {
	log.Printf("[NOTIFICATION] Type: %s | To: %s | Message: %s | Data: %+v", event.Type, event.To, event.Message, event.Data)
	return nil
}

type TelegramSender struct {
	BotToken string
	ChatID   string
}

func (t *TelegramSender) Send(event NotificationEvent) error {
	// TODO: Implement Telegram API integration
	log.Printf("[TELEGRAM] Would send to chat %s: %s", t.ChatID, event.Message)
	return nil
}

type EmailSender struct {
	SMTPServer string
	SMTPPort   int
	Username   string
	Password   string
	From       string
}

func (e *EmailSender) Send(event NotificationEvent) error {
	// TODO: Implement e-mail sending logic (SMTP or provider API)
	log.Printf("[EMAIL] Would send to %s: %s", event.To, event.Message)
	return nil
}

type WhatsAppSender struct {
	APIToken string
	PhoneID  string
}

func (w *WhatsAppSender) Send(event NotificationEvent) error {
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", w.PhoneID)
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                event.To, // deve ser o número com DDI
		"type":              "text",
		"text":              map[string]string{"body": event.Message},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+w.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("WhatsApp API error: %s", resp.Status)
	}
	return nil
}
