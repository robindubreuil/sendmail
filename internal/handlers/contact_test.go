package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"gosendmail/internal/config"
	"gosendmail/internal/models"
	"gosendmail/internal/util"
)

// MockEmailService implements EmailServiceInterface for testing
type MockEmailService struct {
	sendError error
	sentForm  *models.ContactForm
}

func (m *MockEmailService) SendContactForm(ctx context.Context, form *models.ContactForm) error {
	m.sentForm = form
	return m.sendError
}

func (m *MockEmailService) ValidateConfiguration() error {
	return nil
}

func (m *MockEmailService) HealthCheck() error {
	return nil
}

// MockRecaptchaService implements RecaptchaServiceInterface for testing
type MockRecaptchaService struct {
	verifyError error
}

func (m *MockRecaptchaService) Verify(ctx context.Context, response, remoteIP string) error {
	return m.verifyError
}

// MockNonceService implements NonceServiceInterface for testing
type MockNonceService struct {
	validNonce string
}

func (m *MockNonceService) Generate() (string, error) {
	return "test_nonce_123", nil
}

func (m *MockNonceService) Validate(nonce string) bool {
	return nonce == m.validNonce || nonce == "test_nonce_123"
}

func (m *MockNonceService) Shutdown() {}

func createTestConfig() *config.Config {
	return &config.Config{
		Security: config.SecurityConfig{
			MaxMessageSize: 4096,
		},
	}
}

func TestNewContactHandler(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()

	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	if handler.emailService != mockEmail {
		t.Errorf("Expected emailService to be set correctly")
	}

	if handler.recaptchaService != mockRecaptcha {
		t.Errorf("Expected recaptchaService to be set correctly")
	}

	if handler.nonceService != mockNonce {
		t.Errorf("Expected nonceService to be set correctly")
	}

	if handler.config != cfg {
		t.Errorf("Expected config to be set correctly")
	}
}

func TestHandleContact_Success(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success to be true, got false")
	}

	if mockEmail.sentForm == nil {
		t.Errorf("Expected email to be sent")
	}

	if mockEmail.sentForm.FirstName != "Jean" {
		t.Errorf("Expected FirstName 'Jean', got '%s'", mockEmail.sentForm.FirstName)
	}
}

func TestHandleContact_WrongMethod(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	req := httptest.NewRequest(http.MethodGet, "/contact", nil)
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Validation des données échouée" && response.Message != "Données de formulaire invalides" {
		t.Errorf("Expected error message, got '%s'", response.Message)
	}
}

func TestHandleContact_InvalidNonce(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=valid_token&form_nonce=invalid_nonce&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Formulaire invalide ou expiré. Veuillez réessayer." {
		t.Errorf("Expected message about invalid form, got '%s'", response.Message)
	}
}

func TestHandleContact_MissingConsent(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Validation des données échouée" {
		t.Errorf("Expected validation error message, got '%s'", response.Message)
	}
}

func TestHandleContact_ValidationFailure(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Validation des données échouée" {
		t.Errorf("Expected message 'Validation des données échouée', got '%s'", response.Message)
	}

	if response.Errors == nil {
		t.Errorf("Expected validation errors")
	}
}

func TestHandleContact_RecaptchaFailure(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{verifyError: ErrRecaptchaFailed}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=invalid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "La validation reCAPTCHA a échoué" {
		t.Errorf("Expected message 'La validation reCAPTCHA a échoué', got '%s'", response.Message)
	}

	if mockEmail.sentForm != nil {
		t.Errorf("Expected email not to be sent when reCAPTCHA fails")
	}
}

func TestHandleContact_EmailFailure(t *testing.T) {
	mockEmail := &MockEmailService{sendError: ErrSMTPConnection}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Une erreur est survenue lors de l'envoi du message" {
		t.Errorf("Expected message 'Une erreur est survenue lors de l'envoi du message', got '%s'", response.Message)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		remoteAddr     string
		trustedProxies []string
		expectedIP     string
	}{
		{
			name: "X-Forwarded-For with trusted proxy",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			remoteAddr:     "10.0.0.1:12345",
			trustedProxies: []string{"10.0.0.1"},
			expectedIP:     "192.168.1.1",
		},
		{
			name: "X-Real-IP with trusted proxy",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr:     "10.0.0.1:12345",
			trustedProxies: []string{"10.0.0.1"},
			expectedIP:     "192.168.1.2",
		},
		{
			name:           "RemoteAddr fallback no trusted proxies",
			headers:        map[string]string{},
			remoteAddr:     "192.168.1.3:12345",
			trustedProxies: nil,
			expectedIP:     "192.168.1.3",
		},
		{
			name: "X-Forwarded-For ignored without trusted proxy",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.4",
			},
			remoteAddr:     "127.0.0.1:12345",
			trustedProxies: nil,
			expectedIP:     "127.0.0.1",
		},
		{
			name: "X-Forwarded-For with wildcard trusted proxy",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.5",
			},
			remoteAddr:     "10.0.0.1:12345",
			trustedProxies: []string{"*"},
			expectedIP:     "192.168.1.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/contact", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := util.GetClientIP(req, tt.trustedProxies)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestSendJSONSuccessresponse(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	w := httptest.NewRecorder()
	handler.sendJSONSuccessResponse(w, "Test success message")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedContentType := "application/json"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success to be true, got false")
	}

	if response.Message != "Test success message" {
		t.Errorf("Expected message 'Test success message', got '%s'", response.Message)
	}
}

func TestSendJSONErrorresponse(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	errors := []models.ValidationError{
		{Field: "firstName", Message: "Required field"},
		{Field: "email", Message: "Invalid format"},
	}

	w := httptest.NewRecorder()
	handler.sendJSONErrorResponse(w, http.StatusBadRequest, "Test error message", errors)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	expectedContentType := "application/json"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got '%s'", response.Message)
	}

	if response.Errors == nil {
		t.Errorf("Expected errors to be set")
	}
}

func TestSendJSONErrorresponse_WithNilErrors(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	w := httptest.NewRecorder()
	handler.sendJSONErrorResponse(w, http.StatusInternalServerError, "Server error", nil)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Server error" {
		t.Errorf("Expected message 'Server error', got '%s'", response.Message)
	}

	if response.Errors != nil {
		t.Errorf("Expected errors to be nil")
	}
}

func TestHandleContact_WithOptionalFields(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Marie&name=Martin&street=Avenue+des+Champs-Élysées&city=Paris&zip=75008&phone=0987654321&email=&message=Bonjour,+je+veux+vous+contacter&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success to be true, got false")
	}

	if mockEmail.sentForm == nil {
		t.Errorf("Expected email to be sent")
	}

	if mockEmail.sentForm.FirstName != "Marie" {
		t.Errorf("Expected FirstName 'Marie', got '%s'", mockEmail.sentForm.FirstName)
	}

	if mockEmail.sentForm.Email != "" {
		t.Errorf("Expected Email to be empty, got '%s'", mockEmail.sentForm.Email)
	}
}

// Mock errors for testing
var (
	ErrRecaptchaFailed = fmt.Errorf("reCAPTCHA verification failed")
	ErrSMTPConnection  = fmt.Errorf("SMTP connection failed")
)

func TestHandleContact_LargeFormSubmission(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	longMessage := strings.Repeat("This is a very long message. ", 500)

	formData := strings.NewReader(
		fmt.Sprintf("firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=%s&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
			url.QueryEscape(longMessage)),
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactJSON(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for message too long, got %d", resp.StatusCode)
	}

	var response response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success to be false, got true")
	}

	if response.Message != "Validation des données échouée" {
		t.Errorf("Expected validation error message, got '%s'", response.Message)
	}
}

func TestHandleContactHTML_Success(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/sendmail", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactHTML(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedContentType := "text/html; charset=utf-8"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseStr := string(body[:n])

	if !strings.Contains(responseStr, "<!DOCTYPE html>") {
		t.Errorf("Expected HTML response, got: %s", responseStr[:min(100, len(responseStr))])
	}

	if !strings.Contains(responseStr, "Message envoyé avec succès") {
		t.Errorf("Expected success message in HTML response")
	}

	if mockEmail.sentForm == nil {
		t.Errorf("Expected email to be sent")
	}
}

func TestHandleContactHTML_Error(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{verifyError: fmt.Errorf("reCAPTCHA failed")}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Jean&name=Dupont&street=123+Rue+de+la+Paix&city=Paris&zip=75001&phone=0123456789&email=jean.dupont@example.com&message=Test+message&g-recaptcha-response=invalid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/sendmail", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleContactHTML(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	expectedContentType := "text/html; charset=utf-8"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseStr := string(body[:n])

	if !strings.Contains(responseStr, "<!DOCTYPE html>") {
		t.Errorf("Expected HTML response, got: %s", responseStr[:min(100, len(responseStr))])
	}

	if !strings.Contains(responseStr, "Erreur lors de l'envoi") {
		t.Errorf("Expected error message in HTML response")
	}
}

func TestProcessContactForm_WithSpecialCharacters(t *testing.T) {
	mockEmail := &MockEmailService{}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=%C3%89milie&name=L%C3%A9vy&street=15+Avenue+de+l%27Universit%C3%A9&city=Montr%C3%A9al&zip=75001&phone=0123456789&email=emilie.levy%40example.com&message=Message+avec+accents%3A+%C3%A9%C3%A0%C3%B9%C3%AA%C3%AE%C3%B4%C3%BB+et+caract%C3%A8res+sp%C3%A9ciaux%3A+%40%23%24%25%26*&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	form, err := handler.processContactForm(recorder, req)
	if err != nil {
		t.Fatalf("Unexpected error processing form: %v", err)
	}

	if form == nil {
		t.Error("Expected form to be returned")
	}

	if mockEmail.sentForm == nil {
		t.Errorf("Expected email to be sent")
	}

	if mockEmail.sentForm.FirstName != "Émilie" {
		t.Errorf("Expected FirstName 'Émilie', got '%s'", mockEmail.sentForm.FirstName)
	}

	if !strings.Contains(mockEmail.sentForm.Message, "éàùêîôû") {
		t.Error("Expected accents to be preserved in message")
	}
}

func TestProcessContactForm_WithEmailError(t *testing.T) {
	mockEmail := &MockEmailService{sendError: fmt.Errorf("SMTP error")}
	mockRecaptcha := &MockRecaptchaService{}
	mockNonce := &MockNonceService{validNonce: "valid_nonce_123"}
	cfg := createTestConfig()
	handler := NewContactHandler(mockEmail, mockRecaptcha, mockNonce, cfg)

	formData := strings.NewReader(
		"firstName=Test&name=User&street=Test+Street&city=Test+City&zip=12345&phone=0123456789&message=Test+message&g-recaptcha-response=valid_token&form_nonce=valid_nonce_123&invalidCheck=on",
	)

	req := httptest.NewRequest(http.MethodPost, "/contact", formData)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	form, err := handler.processContactForm(recorder, req)
	if err == nil {
		t.Error("Expected error when email sending fails")
	}

	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", err.StatusCode)
	}

	if form != nil {
		t.Error("Expected nil form when email sending fails")
	}
}
