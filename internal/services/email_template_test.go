package services

import (
	"strings"
	"testing"
	"time"

	"gosendmail/internal/models"
)

func TestNewEmailTemplateData(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "  Jean  ",
		Name:      "Dupont",
		Postbox:   "  42  ",
		Street:    "123 Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean.dupont@example.com",
		Message:   "  Ceci est un message de test.  ",
	}

	data := NewEmailTemplateData(form)

	if data.FirstName != "Jean" {
		t.Errorf("Expected FirstName 'Jean', got '%s'", data.FirstName)
	}

	if data.Postbox != "42" {
		t.Errorf("Expected Postbox '42', got '%s'", data.Postbox)
	}

	if data.Message != "Ceci est un message de test." {
		t.Errorf("Expected Message 'Ceci est un message de test.', got '%s'", data.Message)
	}

	if data.Name != "Dupont" {
		t.Errorf("Expected Name 'Dupont', got '%s'", data.Name)
	}

	if data.Zip != "75001" {
		t.Errorf("Expected Zip '75001', got '%s'", data.Zip)
	}

	expectedTime := time.Now().Format("2 January 2006 à 15:04:05")
	if data.Timestamp != expectedTime {
		t.Errorf("Expected timestamp format '%s', got '%s'", expectedTime, data.Timestamp)
	}
}

func TestNewEmailTemplateData_WithNilForm(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when creating template data from nil form")
		}
	}()

	NewEmailTemplateData(nil)
}

func TestGenerateEmailContent_WithValidTemplate(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "Émilie",
		Name:      "Lévy",
		Street:    "15 Avenue des Champs-Élysées",
		City:      "Paris",
		Zip:       "75008",
		Phone:     "0123456789",
		Email:     "emilie@example.com",
		Message:   "Bonjour,\n\nCeci est un test avec des accents: éàùêîôû.",
	}

	content, err := GenerateEmailContent(form)
	if err != nil {
		t.Fatalf("Unexpected error generating email content: %v", err)
	}

	if content.HTML == "" {
		t.Error("Expected HTML content to be non-empty")
	}

	if !strings.Contains(content.HTML, "Émilie Lévy") {
		t.Error("Expected HTML to contain full name")
	}

	if !strings.Contains(content.HTML, "15 Avenue des Champs-Élysées") {
		t.Error("Expected HTML to contain street address")
	}

	if !strings.Contains(content.HTML, "éàùêîôû") {
		t.Error("Expected HTML to preserve accents")
	}

	if content.PlainText == "" {
		t.Error("Expected plain text content to be non-empty")
	}

	if !strings.Contains(content.PlainText, "Nouveau message de contact:") {
		t.Error("Expected plain text to contain header")
	}

	if !strings.Contains(content.PlainText, "Nom complet: Émilie Lévy") {
		t.Error("Expected plain text to contain full name")
	}

	if !strings.Contains(content.PlainText, "Adresse: 15 Avenue des Champs-Élysées") {
		t.Error("Expected plain text to contain street address")
	}
}

func TestGenerateEmailContent_WithoutOptionalFields(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "Marie",
		Name:      "Martin",
		Street:    "Rue de la République",
		City:      "Lyon",
		Zip:       "69001",
		Phone:     "0987654321",
		Email:     "",
		Message:   "Test message without optional fields.",
	}

	content, err := GenerateEmailContent(form)
	if err != nil {
		t.Fatalf("Unexpected error generating email content: %v", err)
	}

	if strings.Contains(content.HTML, "E-mail:") {
		t.Error("HTML should not contain empty optional email field")
	}

	if strings.Contains(content.PlainText, "E-mail:") {
		t.Error("Plain text should not contain empty optional email field")
	}

	if !strings.Contains(content.PlainText, "Nom complet: Marie Martin") {
		t.Error("Expected plain text to contain full name")
	}
}

func TestGenerateEmailContent_WithPostbox(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "Jean",
		Name:      "Dupont",
		Postbox:   "42",
		Street:    "Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean@example.com",
		Message:   "Test message.",
	}

	content, err := GenerateEmailContent(form)
	if err != nil {
		t.Fatalf("Unexpected error generating email content: %v", err)
	}

	if !strings.Contains(content.HTML, "42") {
		t.Error("Expected HTML to contain postbox number")
	}

	if !strings.Contains(content.PlainText, "Boîte postale: 42") {
		t.Error("Expected plain text to contain postbox")
	}
}

func TestGenerateEmailContent_WithoutPostbox(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "Jean",
		Name:      "Dupont",
		Street:    "Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean@example.com",
		Message:   "Test message.",
	}

	content, err := GenerateEmailContent(form)
	if err != nil {
		t.Fatalf("Unexpected error generating email content: %v", err)
	}

	if strings.Contains(content.PlainText, "Boîte postale:") {
		t.Error("Plain text should not contain postbox field when empty")
	}
}

func TestGeneratePlainTextContent(t *testing.T) {
	data := EmailTemplateData{
		FirstName: "Jean",
		Name:      "Dupont",
		Street:    "123 Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Email:     "jean.dupont@example.com",
		Message:   "Ceci est un message de test.\nAvec plusieurs lignes.",
		Timestamp: "12 octobre 2025 à 15:30:45",
	}

	result := generatePlainTextContent(data)

	expectedParts := []string{
		"Nouveau message de contact:",
		"Nom complet: Jean Dupont",
		"Adresse: 123 Rue de la Paix",
		"Code postal: 75001 Paris",
		"Téléphone: 0123456789",
		"E-mail: jean.dupont@example.com",
		"Message:",
		"Ceci est un message de test.",
		"Date d'envoi: 12 octobre 2025 à 15:30:45",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected part '%s' not found in result", part)
		}
	}
}

func TestGeneratePlainTextContent_WithoutOptionalFields(t *testing.T) {
	data := EmailTemplateData{
		FirstName: "Marie",
		Name:      "Martin",
		Street:    "Avenue des Champs-Élysées",
		City:      "Paris",
		Zip:       "75008",
		Phone:     "0987654321",
		Email:     "",
		Message:   "Message sans champs optionnels.",
		Timestamp: "12 octobre 2025 à 16:00:00",
	}

	result := generatePlainTextContent(data)

	expectedRequiredParts := []string{
		"Nouveau message de contact:",
		"Nom complet: Marie Martin",
		"Adresse: Avenue des Champs-Élysées",
		"Code postal: 75008 Paris",
		"Téléphone: 0987654321",
		"Message:",
		"Message sans champs optionnels.",
	}

	for _, part := range expectedRequiredParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected required part '%s' not found in result", part)
		}
	}

	if strings.Contains(result, "E-mail:") {
		t.Errorf("Unexpected empty field 'E-mail:' found in result")
	}
}

func TestGeneratePlainTextContent_WithSpecialCharacters(t *testing.T) {
	data := EmailTemplateData{
		FirstName: "Émilie",
		Name:      "Lévy",
		Street:    "15 Avenue de l'Université",
		City:      "Montréal",
		Zip:       "H3C 3A7",
		Phone:     "0123456789",
		Email:     "emilie.lévy@example.com",
		Message:   "Message avec accents: éàùêîôû et caractères spéciaux: @#$%&*",
		Timestamp: "12 octobre 2025 à 16:30:00",
	}

	result := generatePlainTextContent(data)

	specialChars := []string{
		"Émilie", "Lévy", "Montréal", "éàùêîôû", "@#$%&*",
	}

	for _, char := range specialChars {
		if !strings.Contains(result, char) {
			t.Errorf("Expected special character '%s' not preserved in result", char)
		}
	}
}

func TestGenerateEmailContent_EmbeddedTemplateLoads(t *testing.T) {
	form := &models.ContactForm{
		FirstName: "Test",
		Name:      "User",
		Street:    "Test Street",
		City:      "Test City",
		Zip:       "12345",
		Phone:     "0123456789",
		Message:   "Test message",
	}

	content, err := GenerateEmailContent(form)
	if err != nil {
		t.Fatalf("Embedded template should always load: %v", err)
	}

	if content.HTML == "" {
		t.Error("Expected non-empty HTML from embedded template")
	}
}
