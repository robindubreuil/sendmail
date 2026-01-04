package models

import (
	"strings"
	"testing"
)

func TestContactForm_Validate(t *testing.T) {
	tests := []struct {
		name     string
		form     ContactForm
		expected bool
	}{
		{
			name: "Valid form with all fields",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "123 Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Email:     "jean.dupont@example.com",
				Message:   "Ceci est un message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Valid form without optional fields",
			form: ContactForm{
				FirstName: "Marie",
				Name:      "Martin",
				Street:    "Avenue des Champs-Élysées",
				City:      "Paris",
				Zip:       "75008",
				Phone:     "0987654321",
				Email:     "",
				Message:   "Bonjour, je souhaite vous contacter.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - missing required firstName",
			form: ContactForm{
				FirstName: "",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid zip code",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "1234",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid phone number",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - missing recaptcha",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - missing consent",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   false,
			},
			expected: false,
		},
	}

	maxSize := 4096
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.form.Validate(maxSize)
			if result.IsValid != tt.expected {
				t.Errorf("Expected validation result to be %v, got %v", tt.expected, result.IsValid)
				if !result.IsValid {
					t.Logf("Validation errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestContactForm_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		form     ContactForm
		expected bool
	}{
		{
			name: "Valid form with French characters",
			form: ContactForm{
				FirstName: "Émilie",
				Name:      "Lévy",
				Street:    "15 Rue de l'Université",
				City:      "Saint-Étienne",
				Zip:       "42000",
				Phone:     "0477123456",
				Email:     "emilie.le@example.fr",
				Message:   "Bonjour, je souhaite un rendez-vous.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - empty required name field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - empty street field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - empty city field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - empty zip field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - empty phone field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - empty message field",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - message too long",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   string(make([]byte, 4097)),
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Valid - street with numbers (French address format)",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "123 Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - invalid city (contains numbers)",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris 75001",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid phone (not 10 digits)",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "012345678",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid phone (starts with 1)",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "1123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid email format",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Email:     "invalid-email",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Invalid - invalid name (contains numbers)",
			form: ContactForm{
				FirstName: "Jean123",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
	}

	maxSize := 4096
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.form.Validate(maxSize)
			if result.IsValid != tt.expected {
				t.Errorf("Expected validation result to be %v, got %v", tt.expected, result.IsValid)
				if !result.IsValid {
					t.Logf("Validation errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestContactForm_Validate_FieldLengths(t *testing.T) {
	tests := []struct {
		name     string
		form     ContactForm
		expected bool
	}{
		{
			name: "Valid - long but reasonable name",
			form: ContactForm{
				FirstName: "AlexandreÉdouard",
				Name:      "Dupont",
				Street:    "Rue de la République",
				City:      "SaintÉtienne",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - empty firstName",
			form: ContactForm{
				FirstName: "",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Valid - special French characters",
			form: ContactForm{
				FirstName: "Émilie",
				Name:      "Lévy",
				Street:    "Avenue de l'Université",
				City:      "Saint-Jean-d'Angély",
				Zip:       "17400",
				Phone:     "0546123456",
				Message:   "Bonjour, je souhaite un rendez-vous.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - firstName with numbers",
			form: ContactForm{
				FirstName: "Jean123",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
		{
			name: "Valid - street with numbers",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "123 Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Message de test.",
				Recaptcha: "valid_recaptcha_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
	}

	maxSize := 4096
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.form.Validate(maxSize)
			if result.IsValid != tt.expected {
				t.Errorf("Expected validation result to be %v, got %v", tt.expected, result.IsValid)
				if !result.IsValid {
					t.Logf("Validation errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "Invalid email format",
	}

	expected := "validation error on field 'email': Invalid email format"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   ValidationResult
		expected bool
	}{
		{
			name: "No errors",
			result: ValidationResult{
				IsValid: true,
				Errors:  []ValidationError{},
			},
			expected: false,
		},
		{
			name: "Has errors",
			result: ValidationResult{
				IsValid: false,
				Errors: []ValidationError{
					{Field: "email", Message: "Invalid format"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErrors := len(tt.result.Errors) > 0
			if hasErrors != tt.expected {
				t.Errorf("Expected HasErrors to be %v, got %v", tt.expected, hasErrors)
			}
		})
	}
}

func TestValidationEdgeCases_MultipleErrors(t *testing.T) {
	form := ContactForm{
		FirstName: "",
		Name:      "123",
		Street:    "",
		City:      "",
		Zip:       "123",
		Phone:     "123",
		Email:     "invalid",
		Message:   "",
		Recaptcha: "",
		Nonce:     "valid_nonce",
		Consent:   false,
	}

	maxSize := 4096
	result := form.Validate(maxSize)

	if result.IsValid {
		t.Error("Expected form to be invalid with multiple errors")
	}

	errorCount := len(result.Errors)
	if errorCount < 5 {
		t.Errorf("Expected at least 5 validation errors, got %d", errorCount)
	}

	errorFields := make(map[string]bool)
	for _, err := range result.Errors {
		errorFields[err.Field] = true
	}

	expectedErrorFields := []string{"firstName", "name", "street", "city", "zip", "phone", "message", "recaptcha", "consent"}
	for _, field := range expectedErrorFields {
		if !errorFields[field] {
			t.Errorf("Expected validation error for field '%s', but not found", field)
		}
	}
}

func TestValidationEdgeCases_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name     string
		form     ContactForm
		expected bool
	}{
		{
			name: "Valid - reasonable name length",
			form: ContactForm{
				FirstName: "Jean Michel Alexandre",
				Name:      "De La Roche Saint-André",
				Street:    "Rue de la Grande Armée",
				City:      "Saint-Jean-de-Luz",
				Zip:       "64500",
				Phone:     "0559123456",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Valid - special characters in French names",
			form: ContactForm{
				FirstName: "François-Xavier",
				Name:      "D'Artagnan De Béarn",
				Street:    "Rue de l'Île-de-France",
				City:      "Le Puy-en-Velay",
				Zip:       "43000",
				Phone:     "0471123456",
				Message:   "Message avec accents éàùêîôû",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Invalid - email with domain issues",
			form: ContactForm{
				FirstName: "Jean",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Email:     "jean@.com",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: false,
		},
	}

	maxSize := 4096
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.form.Validate(maxSize)
			if result.IsValid != tt.expected {
				t.Errorf("Expected validation result to be %v, got %v", tt.expected, result.IsValid)
				if !result.IsValid {
					t.Logf("Validation errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidationEdgeCases_RealWorldFrenchNames(t *testing.T) {
	tests := []struct {
		name     string
		form     ContactForm
		expected bool
	}{
		{
			name: "Valid - Jean-Michel Dupont",
			form: ContactForm{
				FirstName: "Jean-Michel",
				Name:      "Dupont",
				Street:    "Rue de la Paix",
				City:      "Paris",
				Zip:       "75001",
				Phone:     "0123456789",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Valid - Marie-Christine D'Angély",
			form: ContactForm{
				FirstName: "Marie-Christine",
				Name:      "D'Angély",
				Street:    "Avenue du Général de Gaulle",
				City:      "Saint-Étienne",
				Zip:       "42000",
				Phone:     "0477123456",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Valid - François-Marie Auber",
			form: ContactForm{
				FirstName: "François-Marie",
				Name:      "Auber",
				Street:    "Place de la Concorde",
				City:      "Paris",
				Zip:       "75008",
				Phone:     "0142123456",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
		{
			name: "Valid - Pierre Henri L'Enfant",
			form: ContactForm{
				FirstName: "Pierre Henri",
				Name:      "L'Enfant",
				Street:    "Boulevard Saint-Michel",
				City:      "Paris",
				Zip:       "75005",
				Phone:     "0144123456",
				Message:   "Test message",
				Recaptcha: "valid_token",
				Nonce:     "valid_nonce",
				Consent:   true,
			},
			expected: true,
		},
	}

	maxSize := 4096
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.form.Validate(maxSize)
			if result.IsValid != tt.expected {
				t.Errorf("Expected validation result to be %v, got %v", tt.expected, result.IsValid)
				if !result.IsValid {
					t.Logf("Validation errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidationEdgeCases_MessageSizeBoundary(t *testing.T) {
	exactlyMaxSize := 4096
	messageAtBoundary := strings.Repeat("A", exactlyMaxSize)

	form := ContactForm{
		FirstName: "Jean",
		Name:      "Dupont",
		Street:    "Rue de la Paix",
		City:      "Paris",
		Zip:       "75001",
		Phone:     "0123456789",
		Message:   messageAtBoundary,
		Recaptcha: "valid_token",
		Nonce:     "valid_nonce",
		Consent:   true,
	}

	maxSize := 4096
	result := form.Validate(maxSize)

	if !result.IsValid {
		t.Errorf("Expected message exactly at max size to be valid, got errors: %+v", result.Errors)
	}

	messageOverBoundary := messageAtBoundary + "X"
	form.Message = messageOverBoundary
	result = form.Validate(maxSize)

	if result.IsValid {
		t.Error("Expected message over max size to be invalid")
	}

	foundSizeError := false
	for _, err := range result.Errors {
		if err.Field == "message" && strings.Contains(err.Message, "4096") {
			foundSizeError = true
			break
		}
	}
	if !foundSizeError {
		t.Error("Expected validation error about message size limit")
	}
}

func TestPhoneValidation_Formats(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected bool
	}{
		{"Valid - standard format", "0123456789", true},
		{"Valid - with spaces", "06 12 34 56 78", true},
		{"Valid - with dots", "06.12.34.56.78", true},
		{"Valid - with dashes", "06-12-34-56-78", true},
		{"Valid - +33 format", "+33612345678", true},
		{"Invalid - starts with 1", "1123456789", false},
		{"Invalid - too short", "012345678", false},
		{"Invalid - too long", "01234567890", false},
		{"Invalid - empty", "", false},
		{"Invalid - letters only", "abcdefghij", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPhone(tt.phone)
			if result != tt.expected {
				t.Errorf("Expected phone %q to be %v, got %v", tt.phone, tt.expected, result)
			}
		})
	}
}
