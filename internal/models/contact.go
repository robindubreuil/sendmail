package models

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	nameRegex    = regexp.MustCompile(`^[a-zA-ZÀ-ÿ]+([a-zA-ZÀ-ÿ'\-\s]*[a-zA-ZÀ-ÿ])?$`)
	streetRegex  = regexp.MustCompile(`^[a-zA-Z0-9À-ÿ'\-\s]+$`)
	zipRegex     = regexp.MustCompile(`^[0-9]{5}$`)
	phoneRegex   = regexp.MustCompile(`^0[1-9][0-9]{8}$`)
	postboxRegex = regexp.MustCompile(`^[0-9]{1,5}$`)
)

type ContactForm struct {
	FirstName string `json:"firstName" form:"firstName"`
	Name      string `json:"name" form:"name"`
	Postbox   string `json:"postbox" form:"postbox"`
	Street    string `json:"street" form:"street"`
	City      string `json:"city" form:"city"`
	Zip       string `json:"zip" form:"zip"`
	Phone     string `json:"phone" form:"phone"`
	Email     string `json:"email" form:"email"`
	Message   string `json:"message" form:"message"`
	Recaptcha string `json:"g-recaptcha-response" form:"g-recaptcha-response"`
	Nonce     string `json:"form_nonce" form:"form_nonce"`
	Consent   bool   `json:"consent" form:"invalidCheck"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

func (cf *ContactForm) Validate(maxMessageSize int) ValidationResult {
	var errors []ValidationError

	if strings.TrimSpace(cf.FirstName) == "" {
		errors = append(errors, ValidationError{
			Field:   "firstName",
			Message: "Le prénom est requis",
		})
	} else if !isValidName(cf.FirstName) {
		errors = append(errors, ValidationError{
			Field:   "firstName",
			Message: "Le prénom n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Le nom est requis",
		})
	} else if !isValidName(cf.Name) {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Le nom n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Postbox) != "" && !isValidPostbox(cf.Postbox) {
		errors = append(errors, ValidationError{
			Field:   "postbox",
			Message: "Le numéro de boîte postale n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Street) == "" {
		errors = append(errors, ValidationError{
			Field:   "street",
			Message: "La rue est requise",
		})
	} else if !isValidStreet(cf.Street) {
		errors = append(errors, ValidationError{
			Field:   "street",
			Message: "La rue n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.City) == "" {
		errors = append(errors, ValidationError{
			Field:   "city",
			Message: "La ville est requise",
		})
	} else if !isValidCity(cf.City) {
		errors = append(errors, ValidationError{
			Field:   "city",
			Message: "La ville n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Zip) == "" {
		errors = append(errors, ValidationError{
			Field:   "zip",
			Message: "Le code postal est requis",
		})
	} else if !isValidZip(cf.Zip) {
		errors = append(errors, ValidationError{
			Field:   "zip",
			Message: "Le code postal n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Phone) == "" {
		errors = append(errors, ValidationError{
			Field:   "phone",
			Message: "Le téléphone est requis",
		})
	} else if !isValidPhone(cf.Phone) {
		errors = append(errors, ValidationError{
			Field:   "phone",
			Message: "Le téléphone n'est pas valide",
		})
	}

	if cf.Email != "" && !isValidEmail(cf.Email) {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "L'adresse e-mail n'est pas valide",
		})
	}

	if strings.TrimSpace(cf.Message) == "" {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Le message est requis",
		})
	} else if utf8.RuneCountInString(cf.Message) > maxMessageSize {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: fmt.Sprintf("Le message ne peut pas dépasser %d caractères", maxMessageSize),
		})
	}

	if strings.TrimSpace(cf.Recaptcha) == "" {
		errors = append(errors, ValidationError{
			Field:   "recaptcha",
			Message: "La validation reCAPTCHA est requise",
		})
	}

	if !cf.Consent {
		errors = append(errors, ValidationError{
			Field:   "consent",
			Message: "Vous devez accepter les conditions pour continuer",
		})
	}

	return ValidationResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

func isValidName(name string) bool {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) == 0 || utf8.RuneCountInString(name) > 64 {
		return false
	}
	return nameRegex.MatchString(name)
}

func isValidPostbox(postbox string) bool {
	postbox = strings.TrimSpace(postbox)
	if utf8.RuneCountInString(postbox) == 0 || utf8.RuneCountInString(postbox) > 5 {
		return false
	}
	return postboxRegex.MatchString(postbox)
}

func isValidStreet(street string) bool {
	street = strings.TrimSpace(street)
	if utf8.RuneCountInString(street) == 0 || utf8.RuneCountInString(street) > 128 {
		return false
	}
	return streetRegex.MatchString(street)
}

func isValidCity(city string) bool {
	city = strings.TrimSpace(city)
	if utf8.RuneCountInString(city) == 0 || utf8.RuneCountInString(city) > 45 {
		return false
	}
	return nameRegex.MatchString(city)
}

func isValidZip(zip string) bool {
	zip = strings.TrimSpace(zip)
	if len(zip) != 5 {
		return false
	}
	return zipRegex.MatchString(zip)
}

func isValidPhone(phone string) bool {
	phone = strings.TrimSpace(phone)

	normalized := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '.', '-':
			return -1
		default:
			return r
		}
	}, phone)

	if strings.HasPrefix(normalized, "+33") {
		if len(normalized) >= 11 {
			normalized = "0" + normalized[3:]
		} else {
			return false
		}
	}

	return phoneRegex.MatchString(normalized)
}

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}
