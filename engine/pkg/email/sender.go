package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

type Sender interface {
	Send(to, subject, htmlBody string) error
	IsConfigured() bool
}

type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) IsConfigured() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}

func (s *SMTPSender) Send(to, subject, htmlBody string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	from := s.cfg.From
	fromAddr := from
	if idx := strings.Index(from, "<"); idx >= 0 {
		fromAddr = strings.Trim(from[idx:], "<>")
	}

	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.TLS {
		return s.sendTLS(addr, fromAddr, to, msg.String())
	}
	return s.sendPlain(addr, fromAddr, to, msg.String())
}

func (s *SMTPSender) sendTLS(addr, from, to, msg string) error {
	host, _, _ := net.SplitHostPort(addr)

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	if s.cfg.User != "" {
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

func (s *SMTPSender) sendPlain(addr, from, to, msg string) error {
	host, _, _ := net.SplitHostPort(addr)

	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

type NoopSender struct{}

func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

func (n *NoopSender) IsConfigured() bool {
	return false
}

func (n *NoopSender) Send(to, subject, htmlBody string) error {
	return fmt.Errorf("email not configured: would send to %s subject=%q", to, subject)
}
