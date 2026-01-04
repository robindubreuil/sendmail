package services

import (
	"bytes"
	"html/template"
	"net/mail"
	"strings"
	"time"

	"gosendmail/internal/models"
	"gosendmail/internal/templates"
)

type EmailTemplateData struct {
	FirstName string
	Name      string
	Postbox   string
	Street    string
	City      string
	Zip       string
	Phone     string
	Email     string
	Message   string
	Timestamp string
}

type EmailContent struct {
	HTML      string
	PlainText string
}

func NewEmailTemplateData(form *models.ContactForm) EmailTemplateData {
	return EmailTemplateData{
		FirstName: strings.TrimSpace(form.FirstName),
		Name:      strings.TrimSpace(form.Name),
		Postbox:   strings.TrimSpace(form.Postbox),
		Street:    strings.TrimSpace(form.Street),
		City:      strings.TrimSpace(form.City),
		Zip:       strings.TrimSpace(form.Zip),
		Phone:     strings.TrimSpace(form.Phone),
		Email:     strings.TrimSpace(form.Email),
		Message:   strings.TrimSpace(form.Message),
		Timestamp: time.Now().Format("2 January 2006 à 15:04:05"),
	}
}

func GenerateEmailContent(form *models.ContactForm) (*EmailContent, error) {
	data := NewEmailTemplateData(form)

	htmlContent, err := generateHTMLContent(data)
	if err != nil {
		return nil, err
	}

	plainTextContent := generatePlainTextContent(data)

	return &EmailContent{
		HTML:      htmlContent,
		PlainText: plainTextContent,
	}, nil
}

func generateHTMLContent(data EmailTemplateData) (string, error) {
	funcMap := template.FuncMap{
		"safeEmail": func(email string) (string, error) {
			if _, err := mail.ParseAddress(email); err != nil {
				return "", err
			}
			return email, nil
		},
	}

	tmpl, err := template.New("contact-email.html").Funcs(funcMap).ParseFS(templates.FS, "contact-email.html")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generatePlainTextContent(data EmailTemplateData) string {
	var builder strings.Builder

	builder.WriteString("Nouveau message de contact:\n\n")
	builder.WriteString("Nom complet: ")
	builder.WriteString(data.FirstName)
	builder.WriteString(" ")
	builder.WriteString(data.Name)
	builder.WriteString("\n")

	if data.Postbox != "" {
		builder.WriteString("Boîte postale: ")
		builder.WriteString(data.Postbox)
		builder.WriteString("\n")
	}

	builder.WriteString("Adresse: ")
	builder.WriteString(data.Street)
	builder.WriteString("\n")

	builder.WriteString("Code postal: ")
	builder.WriteString(data.Zip)
	builder.WriteString(" ")
	builder.WriteString(data.City)
	builder.WriteString("\n")

	builder.WriteString("Téléphone: ")
	builder.WriteString(data.Phone)
	builder.WriteString("\n")

	if data.Email != "" {
		builder.WriteString("E-mail: ")
		builder.WriteString(data.Email)
		builder.WriteString("\n")
	}

	builder.WriteString("\nMessage:\n")
	builder.WriteString(data.Message)
	builder.WriteString("\n\n")
	builder.WriteString("Date d'envoi: ")
	builder.WriteString(data.Timestamp)

	return builder.String()
}
