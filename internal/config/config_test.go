package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "Valid configuration",
			envVars: map[string]string{
				"RECAPTCHA_SECRET_KEY": "test-secret-key",
				"SMTP_HOST":            "smtp.example.com",
				"SMTP_USERNAME":        "test@example.com",
				"SMTP_PASSWORD":        "test-password",
				"FROM_ADDRESS":         "test@example.com",
				"TO_ADDRESS":           "recipient@example.com",
			},
			wantErr: false,
		},
		{
			name: "Missing required reCAPTCHA key",
			envVars: map[string]string{
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_USERNAME": "test@example.com",
				"SMTP_PASSWORD": "test-password",
				"FROM_ADDRESS":  "test@example.com",
				"TO_ADDRESS":    "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Missing required SMTP config",
			envVars: map[string]string{
				"RECAPTCHA_SECRET_KEY": "test-secret-key",
				"SMTP_USERNAME":        "test@example.com",
				"SMTP_PASSWORD":        "test-password",
				"FROM_ADDRESS":         "test@example.com",
				"TO_ADDRESS":           "recipient@example.com",
			},
			wantErr: true,
		},
		{
			name: "Invalid email format",
			envVars: map[string]string{
				"RECAPTCHA_SECRET_KEY": "test-secret-key",
				"SMTP_HOST":            "smtp.example.com",
				"SMTP_USERNAME":        "test@example.com",
				"SMTP_PASSWORD":        "test-password",
				"FROM_ADDRESS":         "invalid-email",
				"TO_ADDRESS":           "recipient@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"RECAPTCHA_SECRET_KEY", "SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD",
				"FROM_ADDRESS", "TO_ADDRESS", "SERVER_HOST", "SERVER_PORT",
				"SMTP_PORT", "MAX_MESSAGE_SIZE", "RATE_LIMIT_REQUESTS", "GOSENDMAIL_CONFIG_FILE",
			} {
				os.Unsetenv(key)
			}

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg.Recaptcha.SecretKey != tt.envVars["RECAPTCHA_SECRET_KEY"] {
				t.Errorf("Expected Recaptcha.SecretKey %s, got %s", tt.envVars["RECAPTCHA_SECRET_KEY"], cfg.Recaptcha.SecretKey)
			}

			if cfg.Email.SMTPHost != tt.envVars["SMTP_HOST"] {
				t.Errorf("Expected Email.SMTPHost %s, got %s", tt.envVars["SMTP_HOST"], cfg.Email.SMTPHost)
			}

			if cfg.Server.Host != "0.0.0.0" {
				t.Errorf("Expected default Server.Host 0.0.0.0, got %s", cfg.Server.Host)
			}

			if cfg.Server.Port != 8080 {
				t.Errorf("Expected default Server.Port 8080, got %d", cfg.Server.Port)
			}
		})
	}
}

func TestLoadWithCustomValues(t *testing.T) {
	for key := range map[string]string{
		"RECAPTCHA_SECRET_KEY": "", "SMTP_HOST": "", "SMTP_PORT": "",
		"SMTP_USERNAME": "", "SMTP_PASSWORD": "", "FROM_ADDRESS": "",
		"TO_ADDRESS": "", "EMAIL_SUBJECT": "", "SERVER_HOST": "",
		"SERVER_PORT": "", "SERVER_READ_TIMEOUT": "", "SERVER_WRITE_TIMEOUT": "",
		"MAX_MESSAGE_SIZE": "", "RATE_LIMIT_REQUESTS": "", "RATE_LIMIT_WINDOW": "",
		"GOSENDMAIL_CONFIG_FILE": "",
	} {
		os.Unsetenv(key)
	}

	envVars := map[string]string{
		"RECAPTCHA_SECRET_KEY": "custom-secret",
		"SMTP_HOST":            "custom.smtp.com",
		"SMTP_PORT":            "465",
		"SMTP_USERNAME":        "custom@example.com",
		"SMTP_PASSWORD":        "custom-password",
		"FROM_ADDRESS":         "custom@example.com",
		"TO_ADDRESS":           "custom-recipient@example.com",
		"EMAIL_SUBJECT":        "Custom Subject",
		"SERVER_HOST":          "127.0.0.1",
		"SERVER_PORT":          "9000",
		"SERVER_READ_TIMEOUT":  "60s",
		"SERVER_WRITE_TIMEOUT": "120s",
		"MAX_MESSAGE_SIZE":     "8000",
		"RATE_LIMIT_REQUESTS":  "20",
		"RATE_LIMIT_WINDOW":    "2m",
	}

	for key, value := range envVars {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Recaptcha.SecretKey != "custom-secret" {
		t.Errorf("Expected Recaptcha.SecretKey 'custom-secret', got '%s'", cfg.Recaptcha.SecretKey)
	}
	if cfg.Email.SMTPPort != 465 {
		t.Errorf("Expected Email.SMTPPort 465, got %d", cfg.Email.SMTPPort)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected Server.Host '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected Server.Port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 60*time.Second {
		t.Errorf("Expected Server.ReadTimeout 60s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 120*time.Second {
		t.Errorf("Expected Server.WriteTimeout 120s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Security.MaxMessageSize != 8000 {
		t.Errorf("Expected Security.MaxMessageSize 8000, got %d", cfg.Security.MaxMessageSize)
	}
	if cfg.Security.RateLimitRequests != 20 {
		t.Errorf("Expected Security.RateLimitRequests 20, got %d", cfg.Security.RateLimitRequests)
	}
	if cfg.Security.RateLimitWindow != 2*time.Minute {
		t.Errorf("Expected Security.RateLimitWindow 2m, got %v", cfg.Security.RateLimitWindow)
	}
	if cfg.Email.Subject != "Custom Subject" {
		t.Errorf("Expected Email.Subject 'Custom Subject', got '%s'", cfg.Email.Subject)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
					Format: "html",
				},
			},
			wantErr: false,
		},
		{
			name: "Empty reCAPTCHA secret key",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: ""},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "Empty SMTP host",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid from address",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "invalid-email", ToAddress: "recipient@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid to address",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "invalid-email",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid SMTP port (zero)",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 0,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid SMTP port (too high)",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 70000,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
					Format: "html",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid email format (not html or text)",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
					Format: "markdown",
				},
			},
			wantErr: true,
		},
		{
			name: "Valid email format html",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
					Format: "html",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid email format text",
			cfg: &Config{
				Recaptcha: RecaptchaConfig{SecretKey: "test-secret-key"},
				Email: EmailConfig{
					SMTPHost: "smtp.example.com", SMTPPort: 587,
					SMTPUsername: "test@example.com", SMTPPassword: "test-password",
					FromAddress: "test@example.com", ToAddress: "recipient@example.com",
					Format: "text",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	t.Run("Environment variable set", func(t *testing.T) {
		t.Setenv("TEST_VAR", "test_value")
		result := getEnv("TEST_VAR", "default_value")
		if result != "test_value" {
			t.Errorf("Expected test_value, got %s", result)
		}
	})

	t.Run("Environment variable not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR_UNSET")
		result := getEnv("TEST_VAR_UNSET", "default_value")
		if result != "default_value" {
			t.Errorf("Expected default_value, got %s", result)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("Valid integer", func(t *testing.T) {
		t.Setenv("TEST_INT", "123")
		result := getEnvInt("TEST_INT", 456)
		if result != 123 {
			t.Errorf("Expected 123, got %d", result)
		}
	})

	t.Run("Invalid integer", func(t *testing.T) {
		t.Setenv("TEST_INT_INV", "invalid")
		result := getEnvInt("TEST_INT_INV", 456)
		if result != 456 {
			t.Errorf("Expected 456, got %d", result)
		}
	})

	t.Run("Not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT_EMPTY")
		result := getEnvInt("TEST_INT_EMPTY", 456)
		if result != 456 {
			t.Errorf("Expected 456, got %d", result)
		}
	})
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("Valid duration", func(t *testing.T) {
		t.Setenv("TEST_DURATION", "30s")
		result := getEnvDuration("TEST_DURATION", 60*time.Second)
		if result != 30*time.Second {
			t.Errorf("Expected 30s, got %v", result)
		}
	})

	t.Run("Invalid duration", func(t *testing.T) {
		t.Setenv("TEST_DURATION_INV", "invalid")
		result := getEnvDuration("TEST_DURATION_INV", 60*time.Second)
		if result != 60*time.Second {
			t.Errorf("Expected 60s, got %v", result)
		}
	})

	t.Run("Not set", func(t *testing.T) {
		os.Unsetenv("TEST_DURATION_EMPTY")
		result := getEnvDuration("TEST_DURATION_EMPTY", 60*time.Second)
		if result != 60*time.Second {
			t.Errorf("Expected 60s, got %v", result)
		}
	})
}

func TestLoadFromFile(t *testing.T) {
	validConfig := `
server:
  host: "127.0.0.1"
  port: 9000
  read_timeout: 45s
  write_timeout: 90s

recaptcha:
  secret_key: "yaml-secret-key"
  verify_url: "https://custom.recaptcha.url"

email:
  smtp_host: "yaml.smtp.com"
  smtp_port: 465
  smtp_username: "yaml@example.com"
  smtp_password: "yaml-password"
  from_address: "yaml@example.com"
  to_address: "yaml-recipient@example.com"
  subject: "YAML Test Subject"
  format: "text"

security:
  max_message_size: 8192
  rate_limit_requests: 15
  rate_limit_window: 5m
`

	tests := []struct {
		name          string
		configContent string
		envVars       map[string]string
		expectErr     bool
		expectedHost  string
		expectedPort  int
		expectedKey   string
	}{
		{
			name:          "Valid YAML config",
			configContent: validConfig,
			envVars: map[string]string{
				"RECAPTCHA_SECRET_KEY": "env-secret-key",
				"SMTP_HOST":            "env.smtp.com",
			},
			expectErr:    false,
			expectedHost: "127.0.0.1",
			expectedPort: 9000,
			expectedKey:  "env-secret-key",
		},
		{
			name: "Minimal YAML config",
			configContent: `
server:
  port: 8080
`,
			envVars: map[string]string{
				"RECAPTCHA_SECRET_KEY": "test-secret",
				"SMTP_HOST":            "smtp.test.com",
				"SMTP_USERNAME":        "test@test.com",
				"SMTP_PASSWORD":        "test-password",
				"FROM_ADDRESS":         "test@test.com",
				"TO_ADDRESS":           "recipient@test.com",
			},
			expectErr:    false,
			expectedHost: "0.0.0.0",
			expectedPort: 8080,
			expectedKey:  "test-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"RECAPTCHA_SECRET_KEY", "SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD",
				"FROM_ADDRESS", "TO_ADDRESS", "SERVER_HOST", "SERVER_PORT", "GOSENDMAIL_CONFIG_FILE",
			} {
				os.Unsetenv(key)
			}

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configContent); err != nil {
				t.Fatalf("Failed to write config content: %v", err)
			}
			tmpFile.Close()

			t.Setenv("GOSENDMAIL_CONFIG_FILE", tmpFile.Name())

			cfg, err := Load()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg.Server.Host != tt.expectedHost {
				t.Errorf("Expected server host %s, got %s", tt.expectedHost, cfg.Server.Host)
			}
			if cfg.Server.Port != tt.expectedPort {
				t.Errorf("Expected server port %d, got %d", tt.expectedPort, cfg.Server.Port)
			}
			if cfg.Recaptcha.SecretKey != tt.expectedKey {
				t.Errorf("Expected recaptcha secret key %s, got %s", tt.expectedKey, cfg.Recaptcha.SecretKey)
			}
		})
	}
}

func TestLoadFromInvalidFile(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectErr     bool
	}{
		{
			name:          "Invalid YAML syntax",
			configContent: "invalid: yaml: content: [",
			expectErr:     true,
		},
		{
			name:          "Empty file",
			configContent: "",
			expectErr:     false,
		},
		{
			name:          "Non-existent file path",
			configContent: "",
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"RECAPTCHA_SECRET_KEY", "SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD",
				"FROM_ADDRESS", "TO_ADDRESS", "GOSENDMAIL_CONFIG_FILE",
			} {
				os.Unsetenv(key)
			}

			if tt.name == "Non-existent file path" {
				t.Setenv("GOSENDMAIL_CONFIG_FILE", "/non/existent/config.yaml")
			} else {
				tmpFile, err := os.CreateTemp("", "invalid-config-*.yaml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if tt.configContent != "" {
					tmpFile.WriteString(tt.configContent)
				}
				tmpFile.Close()
				t.Setenv("GOSENDMAIL_CONFIG_FILE", tmpFile.Name())
			}

			if !tt.expectErr {
				t.Setenv("RECAPTCHA_SECRET_KEY", "test-secret")
				t.Setenv("SMTP_HOST", "smtp.test.com")
				t.Setenv("SMTP_USERNAME", "test@test.com")
				t.Setenv("SMTP_PASSWORD", "test-password")
				t.Setenv("FROM_ADDRESS", "test@test.com")
				t.Setenv("TO_ADDRESS", "recipient@test.com")
			}

			cfg, err := Load()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg != nil && !tt.expectErr {
				if cfg.Server.Port != 8080 && tt.name != "Non-existent file path" {
					t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
				}
			}
		})
	}
}

func TestConfigFilePrecedence(t *testing.T) {
	configContent := `
server:
  host: "test-host"
  port: 9999
`

	tmpDir, err := os.MkdirTemp("", "sendmail-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localConfig := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localConfig, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write local config: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	for _, key := range []string{
		"RECAPTCHA_SECRET_KEY", "SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD",
		"FROM_ADDRESS", "TO_ADDRESS", "GOSENDMAIL_CONFIG_FILE",
	} {
		os.Unsetenv(key)
	}

	t.Setenv("RECAPTCHA_SECRET_KEY", "test-secret")
	t.Setenv("SMTP_HOST", "smtp.test.com")
	t.Setenv("SMTP_USERNAME", "test@test.com")
	t.Setenv("SMTP_PASSWORD", "test-password")
	t.Setenv("FROM_ADDRESS", "test@test.com")
	t.Setenv("TO_ADDRESS", "recipient@test.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Server.Host != "test-host" {
		t.Errorf("Expected server host 'test-host' from config file, got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected server port 9999 from config file, got %d", cfg.Server.Port)
	}
}
