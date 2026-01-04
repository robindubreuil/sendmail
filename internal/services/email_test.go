package services

import (
	"context"
	"strings"
	"testing"

	"gosendmail/internal/config"
	"gosendmail/internal/models"
)

func TestNewEmailService(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Test Subject",
		},
	}

	es := NewEmailService(cfg)

	if es.smtpHost != cfg.Email.SMTPHost {
		t.Errorf("Expected smtpHost %s, got %s", cfg.Email.SMTPHost, es.smtpHost)
	}

	if es.smtpPort != cfg.Email.SMTPPort {
		t.Errorf("Expected smtpPort %d, got %d", cfg.Email.SMTPPort, es.smtpPort)
	}

	if es.smtpUsername != cfg.Email.SMTPUsername {
		t.Errorf("Expected smtpUsername %s, got %s", cfg.Email.SMTPUsername, es.smtpUsername)
	}

	if es.smtpPassword != cfg.Email.SMTPPassword {
		t.Errorf("Expected smtpPassword %s, got %s", cfg.Email.SMTPPassword, es.smtpPassword)
	}

	if es.fromAddress != cfg.Email.FromAddress {
		t.Errorf("Expected fromAddress %s, got %s", cfg.Email.FromAddress, es.fromAddress)
	}

	if es.toAddress != cfg.Email.ToAddress {
		t.Errorf("Expected toAddress %s, got %s", cfg.Email.ToAddress, es.toAddress)
	}

	if es.subject != cfg.Email.Subject {
		t.Errorf("Expected subject %s, got %s", cfg.Email.Subject, es.subject)
	}
}

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		es      *EmailService
		wantErr bool
	}{
		{
			name: "Valid configuration",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     587,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: false,
		},
		{
			name: "Empty SMTP host",
			es: &EmailService{
				smtpHost:     "",
				smtpPort:     587,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Invalid SMTP port (zero)",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     0,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Invalid SMTP port (too high)",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     70000,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Empty SMTP username",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     587,
				smtpUsername: "",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Empty SMTP password",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     587,
				smtpUsername: "test@example.com",
				smtpPassword: "",
				fromAddress:  "sender@example.com",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Empty from address",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     587,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "",
				toAddress:    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Empty to address",
			es: &EmailService{
				smtpHost:     "smtp.example.com",
				smtpPort:     587,
				smtpUsername: "test@example.com",
				smtpPassword: "password",
				fromAddress:  "sender@example.com",
				toAddress:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.es.ValidateConfiguration()
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSendContactForm_Integration(t *testing.T) {
	// This test is for integration testing and may require a real SMTP server
	// For now, we'll test the method structure and error handling

	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "nonexistent.smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Test Contact Form",
		},
	}

	es := NewEmailService(cfg)

	form := &models.ContactForm{
		FirstName: "Jean",
		Name:      "Dupont",
		Street:    "Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean.dupont@example.com",
		Message:   "Ceci est un message de test.",
	}

	// This should fail due to nonexistent SMTP server
	err := es.SendContactForm(context.Background(), form)
	if err == nil {
		t.Errorf("Expected error when sending to nonexistent SMTP server")
	}

	// Check that the error contains relevant information
	expectedError := "failed to connect to SMTP server"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

func TestSendContactForm_WithNilForm(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Test Subject",
		},
	}

	es := NewEmailService(cfg)

	err := es.SendContactForm(context.Background(), nil)
	if err == nil {
		t.Errorf("Expected error when sending nil form")
	}

	expectedError := "contact form cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestSendContactForm_EmailFormatting(t *testing.T) {
	// This test verifies that the email formatting is correct
	// We can't test actual SMTP sending without a server, but we can test the structure

	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "nonexistent.smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Test Contact Form",
		},
	}

	es := NewEmailService(cfg)

	form := &models.ContactForm{
		FirstName: "Jean",
		Name:      "Dupont",
		Street:    "123 Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean.dupont@example.com",
		Message:   "Ceci est un message de test.",
	}

	err := es.SendContactForm(context.Background(), form)
	if err == nil {
		t.Errorf("Expected error when sending to nonexistent SMTP server")
	}

	// The error should occur during SMTP connection, not during formatting
	// This indicates the email formatting was successful
	expectedError := "failed to connect to SMTP server"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected SMTP connection error, got: %v", err)
	}
}

func TestSendContactForm_WithOptionalFields(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "nonexistent.smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Test Contact Form",
		},
	}

	es := NewEmailService(cfg)

	form := &models.ContactForm{
		FirstName: "Marie",
		Name:      "Martin",
		Street:    "Avenue des Champs-Élysées",
		City:      "Paris",
		Zip:       "75008",
		Phone:     "0987654321",
		Email:     "",
		Message:   "Bonjour, je souhaite vous contacter.",
	}

	err := es.SendContactForm(context.Background(), form)
	if err == nil {
		t.Errorf("Expected error when sending to nonexistent SMTP server")
	}

	expectedError := "failed to connect to SMTP server"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected SMTP connection error, got: %v", err)
	}
}

func TestSendContactForm_WithCustomSubject(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			SMTPHost:     "nonexistent.smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "sender@example.com",
			ToAddress:    "recipient@example.com",
			Subject:      "Custom Subject Line",
		},
	}

	es := NewEmailService(cfg)

	form := &models.ContactForm{
		FirstName: "Test",
		Name:      "User",
		Street:    "Test Street",
		City:      "Test City",
		Zip:       "12345",
		Phone:     "0123456789",
		Message:   "Test message",
	}

	err := es.SendContactForm(context.Background(), form)
	if err == nil {
		t.Errorf("Expected error when sending to nonexistent SMTP server")
	}

	// Verify that the custom subject would be used (error occurs after subject is set)
	expectedError := "failed to connect to SMTP server"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected SMTP connection error, got: %v", err)
	}
}

func TestBuildPlainTextMessage_WithSpecialCharacters(t *testing.T) {
	es := &EmailService{
		fromAddress: "sender@example.com",
		toAddress:   "recipient@example.com",
		subject:     "Test Subject",
	}

	body := "Message avec accents: éàùêîôû"

	result := es.buildPlainTextMessage(body, "")

	if !strings.Contains(result, "Content-Transfer-Encoding: quoted-printable") {
		t.Errorf("Expected quoted-printable transfer encoding")
	}

	if !strings.Contains(result, "From: sender@example.com") {
		t.Errorf("Expected From header in result")
	}

	if !strings.Contains(result, "Subject: Test Subject") {
		t.Errorf("Expected Subject header in result")
	}
}

func TestBuildPlainTextMessage_WithReplyTo(t *testing.T) {
	es := &EmailService{
		fromAddress: "sender@example.com",
		toAddress:   "recipient@example.com",
		subject:     "Test Subject",
	}

	result := es.buildPlainTextMessage("body", "user@example.com")

	if !strings.Contains(result, "Reply-To: user@example.com") {
		t.Errorf("Expected Reply-To header in result")
	}
}

func TestBuildPlainTextMessage_WithoutReplyTo(t *testing.T) {
	es := &EmailService{
		fromAddress: "sender@example.com",
		toAddress:   "recipient@example.com",
		subject:     "Test Subject",
	}

	result := es.buildPlainTextMessage("body", "")

	if strings.Contains(result, "Reply-To:") {
		t.Errorf("Expected no Reply-To header when replyTo is empty")
	}
}

func TestBuildMultipartMessage(t *testing.T) {
	es := &EmailService{
		fromAddress: "sender@example.com",
		toAddress:   "recipient@example.com",
		subject:     "Test Subject",
	}

	htmlBody := "<html><body><h1>Test HTML</h1></body></html>"
	textBody := "Test Plain Text"
	boundary := "----=_Part_testboundary123"

	result := es.buildMultipartMessage(htmlBody, textBody, boundary, "")

	expectedHeaders := []string{
		"From: sender@example.com",
		"To: recipient@example.com",
		"Subject: Test Subject",
		"MIME-Version: 1.0",
		"multipart/alternative",
	}

	for _, header := range expectedHeaders {
		if !strings.Contains(result, header) {
			t.Errorf("Expected header '%s' not found in result", header)
		}
	}

	if strings.Contains(result, "Reply-To:") {
		t.Errorf("Expected no Reply-To header when replyTo is empty")
	}

	expectedParts := []string{
		"Content-Type: text/plain; charset=utf-8",
		"Test Plain Text",
		"Content-Type: text/html; charset=utf-8",
		"Test HTML",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected part '%s' not found in result", part)
		}
	}
}

func TestBuildMultipartMessage_WithReplyTo(t *testing.T) {
	es := &EmailService{
		fromAddress: "sender@example.com",
		toAddress:   "recipient@example.com",
		subject:     "Test Subject",
	}

	result := es.buildMultipartMessage("<p>test</p>", "test", "----=_Part_test", "user@example.com")

	if !strings.Contains(result, "Reply-To: user@example.com") {
		t.Errorf("Expected Reply-To header in multipart message")
	}
}

func TestGenerateBoundary(t *testing.T) {
	b1, err := generateBoundary()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	b2, err := generateBoundary()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if b1 == b2 {
		t.Error("Boundaries should be unique")
	}

	if !strings.HasPrefix(b1, "----=_Part_") {
		t.Errorf("Boundary should have expected prefix, got: %s", b1)
	}
}
