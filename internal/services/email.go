package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"gosendmail/internal/config"
	"gosendmail/internal/models"
)

type EmailService struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromAddress  string
	toAddress    string
	subject      string
	format       string
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		smtpHost:     cfg.Email.SMTPHost,
		smtpPort:     cfg.Email.SMTPPort,
		smtpUsername: cfg.Email.SMTPUsername,
		smtpPassword: cfg.Email.SMTPPassword,
		fromAddress:  cfg.Email.FromAddress,
		toAddress:    cfg.Email.ToAddress,
		subject:      cfg.Email.Subject,
		format:       cfg.Email.Format,
	}
}

func (es *EmailService) SendContactForm(ctx context.Context, form *models.ContactForm) error {
	if form == nil {
		return fmt.Errorf("contact form cannot be nil")
	}

	var message string
	replyTo := strings.TrimSpace(form.Email)
	if es.format == "html" {
		content, err := GenerateEmailContent(form)
		if err != nil {
			return fmt.Errorf("failed to generate email content: %w", err)
		}
		boundary, err := generateBoundary()
		if err != nil {
			return fmt.Errorf("failed to generate boundary: %w", err)
		}
		message = es.buildMultipartMessage(content.HTML, content.PlainText, boundary, replyTo)
	} else {
		message = es.buildPlainTextMessage(generatePlainTextContent(NewEmailTemplateData(form)), replyTo)
	}

	client, err := es.dialSMTP(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", es.smtpUsername, es.smtpPassword, es.smtpHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	if err := client.Mail(es.fromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(es.toAddress); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

func (es *EmailService) dialSMTP(ctx context.Context) (*smtp.Client, error) {
	smtpAddr := fmt.Sprintf("%s:%d", es.smtpHost, es.smtpPort)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	conn, err := dialer.DialContext(ctx, "tcp", smtpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	if es.smtpPort == 465 {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: es.smtpHost})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("failed to perform TLS handshake: %w", err)
		}
		client, err := smtp.NewClient(tlsConn, es.smtpHost)
		if err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}
		return client, nil
	}

	client, err := smtp.NewClient(conn, es.smtpHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		client.Close()
		return nil, fmt.Errorf("STARTTLS is required but not supported by the SMTP server")
	}
	if err := client.StartTLS(&tls.Config{ServerName: es.smtpHost}); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	return client, nil
}

func generateBoundary() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "----=_Part_" + hex.EncodeToString(b), nil
}

func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func encodeSubject(subject string) string {
	return mime.QEncoding.Encode("utf-8", subject)
}

func encodeQuotedPrintable(input string) string {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	w.Write([]byte(input))
	w.Close()
	return buf.String()
}

func (es *EmailService) buildMultipartMessage(htmlBody, textBody, boundary, replyTo string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", sanitizeHeader(es.fromAddress)))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", sanitizeHeader(es.toAddress)))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(sanitizeHeader(es.subject))))
	if replyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", sanitizeHeader(replyTo)))
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(encodeQuotedPrintable(textBody))
	buf.WriteString("\r\n\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(encodeQuotedPrintable(htmlBody))
	buf.WriteString("\r\n\r\n")

	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.String()
}

func (es *EmailService) buildPlainTextMessage(body, replyTo string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", sanitizeHeader(es.fromAddress)))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", sanitizeHeader(es.toAddress)))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(sanitizeHeader(es.subject))))
	if replyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", sanitizeHeader(replyTo)))
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")

	buf.WriteString(encodeQuotedPrintable(body))

	return buf.String()
}

func (es *EmailService) ValidateConfiguration() error {
	if es.smtpHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if es.smtpPort <= 0 || es.smtpPort > 65535 {
		return fmt.Errorf("invalid SMTP port")
	}
	if es.smtpUsername == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if es.smtpPassword == "" {
		return fmt.Errorf("SMTP password is required")
	}
	if es.fromAddress == "" {
		return fmt.Errorf("from address is required")
	}
	if es.toAddress == "" {
		return fmt.Errorf("to address is required")
	}

	if _, err := mail.ParseAddress(es.fromAddress); err != nil {
		return fmt.Errorf("invalid from address format: %w", err)
	}

	if _, err := mail.ParseAddress(es.toAddress); err != nil {
		return fmt.Errorf("invalid to address format: %w", err)
	}

	return nil
}

func (es *EmailService) HealthCheck() error {
	smtpAddr := fmt.Sprintf("%s:%d", es.smtpHost, es.smtpPort)
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", smtpAddr)
	if err != nil {
		return fmt.Errorf("SMTP server unreachable: %w", err)
	}
	conn.Close()
	return nil
}
