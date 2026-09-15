package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"

	"simas-backend/internal/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type SMTPMailer struct {
	cfg *config.MailConfig
}

func NewSMTPMailer(cfg *config.MailConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// SendAsync sends email in a goroutine to avoid blocking API response
func (m *SMTPMailer) SendAsync(to, subject, body string) {
	go func() {
		if err := m.Send(context.Background(), to, subject, body); err != nil {
			log.Printf("mailer: failed to send to %s: %v", maskEmail(to), err)
		}
	}()
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.cfg.Host == "" {
		return fmt.Errorf("mailer: SMTP host not configured")
	}

	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))
	from := m.cfg.From
	if from == "" {
		from = m.cfg.Username
	}

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s",
		m.cfg.FromName, from, to, subject, body)

	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

	if m.cfg.Encryption == "ssl" {
		return sendSSL(addr, from, []string{to}, []byte(msg), auth, m.cfg.Host)
	}
	return sendSTARTTLS(addr, from, []string{to}, []byte(msg), auth, m.cfg.Host)
}

func sendSTARTTLS(addr, from string, to []string, msg []byte, auth smtp.Auth, host string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	tlsCfg := &tls.Config{ServerName: host}
	if err := client.StartTLS(tlsCfg); err != nil {
		return err
	}
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, t := range to {
		if err := client.Rcpt(t); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func sendSSL(addr, from string, to []string, msg []byte, auth smtp.Auth, host string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, t := range to {
		if err := client.Rcpt(t); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func maskEmail(email string) string {
	if len(email) < 4 {
		return "****"
	}
	for i, c := range email {
		if c == '@' {
			if i <= 2 {
				return "**" + email[i:]
			}
			return email[:2] + "**" + email[i:]
		}
	}
	return email[:2] + "****"
}
