package email

import (
	"fmt"
	"net/smtp"
)

// Mailer is the strategy interface for email delivery (OCP: new providers
// implement this without modifying existing code).
type Mailer interface {
	Send(to, subject, htmlBody string) error
}

// SMTPMailer sends mail via an SMTP relay.
type SMTPMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPMailer(host, port, username, password, from string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password, from: from}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	if m.host == "" || m.from == "" {
		return fmt.Errorf("SMTP not configured: host=%q from=%q", m.host, m.from)
	}
	headers := "From: " + m.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n"
	msg := []byte(headers + body)
	addr := m.host + ":" + m.port
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	return smtp.SendMail(addr, auth, m.from, []string{to}, msg)
}

// NoopMailer silently succeeds; used when SMTP is not configured.
type NoopMailer struct{}

func (n *NoopMailer) Send(_, _, _ string) error { return nil }
