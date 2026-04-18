package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"gosendmail/internal/config"
	"gosendmail/internal/middleware"
	"gosendmail/internal/models"
	"gosendmail/internal/services"
	"gosendmail/internal/templates"
	"gosendmail/internal/util"
)

type ContactHandler struct {
	emailService     services.EmailServiceInterface
	recaptchaService services.RecaptchaServiceInterface
	nonceService     services.NonceServiceInterface
	config           *config.Config
	successTemplate  *template.Template
	errorTemplate    *template.Template
}

type response struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Errors  []models.ValidationError `json:"errors,omitempty"`
}

func NewContactHandler(emailService services.EmailServiceInterface, recaptchaService services.RecaptchaServiceInterface, nonceService services.NonceServiceInterface, cfg *config.Config) *ContactHandler {
	tmplFS := templates.FS

	successTemplate, err := template.ParseFS(tmplFS, "success.html")
	if err != nil {
		slog.Error("Failed to load success template", "error", err)
		successTemplate = nil
	}

	errorTemplate, err := template.ParseFS(tmplFS, "error.html")
	if err != nil {
		slog.Error("Failed to load error template", "error", err)
		errorTemplate = nil
	}

	return &ContactHandler{
		emailService:     emailService,
		recaptchaService: recaptchaService,
		nonceService:     nonceService,
		config:           cfg,
		successTemplate:  successTemplate,
		errorTemplate:    errorTemplate,
	}
}

func (h *ContactHandler) HandleContactHTML(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	form, err := h.processContactForm(w, r)
	if err != nil {
		h.sendHTMLErrorResponse(w, err.StatusCode, err.Message, err.Errors)
		return
	}

	log.Info("Contact form submitted successfully",
		"first_name", form.FirstName,
		"name", form.Name,
	)
	h.sendHTMLSuccessResponse(w, "Votre message a été envoyé avec succès")
}

func (h *ContactHandler) HandleContactJSON(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	form, err := h.processContactForm(w, r)
	if err != nil {
		h.sendJSONErrorResponse(w, err.StatusCode, err.Message, err.Errors)
		return
	}

	log.Info("Contact form submitted successfully",
		"first_name", form.FirstName,
		"name", form.Name,
	)
	h.sendJSONSuccessResponse(w, "Votre message a été envoyé avec succès")
}

type formError struct {
	StatusCode int
	Message    string
	Errors     []models.ValidationError
}

func (e *formError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("form error (status %d): %s: %v", e.StatusCode, e.Message, e.Errors)
	}
	return fmt.Sprintf("form error (status %d): %s", e.StatusCode, e.Message)
}

func (h *ContactHandler) processContactForm(w http.ResponseWriter, r *http.Request) (*models.ContactForm, *formError) {
	log := middleware.GetLogger(r.Context())

	maxBodySize := int64(h.config.Security.MaxMessageSize) * 5
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := r.ParseForm(); err != nil {
		log.Warn("Failed to parse form", "error", err)
		return nil, &formError{
			StatusCode: http.StatusBadRequest,
			Message:    "Données de formulaire invalides",
		}
	}

	nonce := r.FormValue("form_nonce")
	if !h.nonceService.Validate(nonce) {
		log.Warn("Invalid or expired nonce")
		return nil, &formError{
			StatusCode: http.StatusBadRequest,
			Message:    "Formulaire invalide ou expiré. Veuillez réessayer.",
		}
	}

	consentValue := r.FormValue("invalidCheck")
	consent := consentValue == "on"

	form := &models.ContactForm{
		FirstName: r.FormValue("firstName"),
		Name:      r.FormValue("name"),
		Postbox:   r.FormValue("postbox"),
		Street:    r.FormValue("street"),
		City:      r.FormValue("city"),
		Zip:       r.FormValue("zip"),
		Phone:     r.FormValue("phone"),
		Email:     r.FormValue("email"),
		Message:   r.FormValue("message"),
		Recaptcha: r.FormValue("g-recaptcha-response"),
		Nonce:     nonce,
		Consent:   consent,
	}

	validationResult := form.Validate(h.config.Security.MaxMessageSize)
	if !validationResult.IsValid {
		log.Warn("Form validation failed", "errors", validationResult.Errors)
		return nil, &formError{
			StatusCode: http.StatusBadRequest,
			Message:    "Validation des données échouée",
			Errors:     validationResult.Errors,
		}
	}

	remoteIP := util.GetClientIP(r, h.config.Security.TrustedProxies)

	if err := h.recaptchaService.Verify(r.Context(), form.Recaptcha, remoteIP); err != nil {
		log.Warn("reCAPTCHA verification failed", "error", err, "remote_ip", remoteIP)
		return nil, &formError{
			StatusCode: http.StatusBadRequest,
			Message:    "La validation reCAPTCHA a échoué",
		}
	}

	if err := h.emailService.SendContactForm(r.Context(), form); err != nil {
		log.Error("Failed to send email", "error", err)
		return nil, &formError{
			StatusCode: http.StatusInternalServerError,
			Message:    "Une erreur est survenue lors de l'envoi du message",
		}
	}

	return form, nil
}

func (h *ContactHandler) sendJSONSuccessResponse(w http.ResponseWriter, message string) {
	resp := response{
		Success: true,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode success response", "error", err)
	}
}

func (h *ContactHandler) sendJSONErrorResponse(w http.ResponseWriter, statusCode int, message string, errors []models.ValidationError) {
	resp := response{
		Success: false,
		Message: message,
		Errors:  errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}

func (h *ContactHandler) sendHTMLSuccessResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	templateData := struct {
		Message string
	}{
		Message: message,
	}

	if h.successTemplate != nil {
		if err := h.successTemplate.Execute(w, templateData); err != nil {
			slog.Error("Failed to execute success template", "error", err)
			writeFallbackSuccessHTML(w, message)
		}
	} else {
		writeFallbackSuccessHTML(w, message)
	}
}

func writeFallbackSuccessHTML(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="fr"><head><meta charset="UTF-8"><title>Message envoyé</title></head>
<body style="font-family:sans-serif;text-align:center;padding:50px">
<h1 style="color:#28a745">Message envoyé avec succès</h1>
<p>%s</p>
<a href="javascript:history.back()">Retour au formulaire</a>
</body></html>`, template.HTMLEscapeString(message))
}

func (h *ContactHandler) sendHTMLErrorResponse(w http.ResponseWriter, statusCode int, message string, errors []models.ValidationError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	templateData := struct {
		Message string
		Errors  []models.ValidationError
	}{
		Message: message,
		Errors:  errors,
	}

	if h.errorTemplate != nil {
		if err := h.errorTemplate.Execute(w, templateData); err != nil {
			slog.Error("Failed to execute error template", "error", err)
			writeFallbackErrorHTML(w, message, errors)
		}
	} else {
		writeFallbackErrorHTML(w, message, errors)
	}
}

func writeFallbackErrorHTML(w http.ResponseWriter, message string, errors []models.ValidationError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html lang="fr"><head><meta charset="UTF-8"><title>Erreur</title></head>
<body style="font-family:sans-serif;text-align:center;padding:50px">
<h1 style="color:#dc3545">Erreur lors de l'envoi</h1>
<p>{{.Message}}</p>
{{if .Errors}}<ul>{{range .Errors}}<li>{{.Message}}</li>{{end}}</ul>{{end}}
<a href="javascript:history.back()">Retour au formulaire</a>
</body></html>`))

	data := struct {
		Message string
		Errors  []models.ValidationError
	}{
		Message: message,
		Errors:  errors,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("Failed to execute fallback error template", "error", err)
	}
}
