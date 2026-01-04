package config

import (
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Recaptcha RecaptchaConfig `yaml:"recaptcha"`
	Email     EmailConfig     `yaml:"email"`
	Security  SecurityConfig  `yaml:"security"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type RecaptchaConfig struct {
	SecretKey      string  `yaml:"secret_key"`
	VerifyURL      string  `yaml:"verify_url"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

type EmailConfig struct {
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	SMTPUsername string `yaml:"smtp_username"`
	SMTPPassword string `yaml:"smtp_password"`
	FromAddress  string `yaml:"from_address"`
	ToAddress    string `yaml:"to_address"`
	Subject      string `yaml:"subject"`
	Format       string `yaml:"format"`
}

type SecurityConfig struct {
	MaxMessageSize    int           `yaml:"max_message_size"`
	RateLimitRequests int           `yaml:"rate_limit_requests"`
	RateLimitWindow   time.Duration `yaml:"rate_limit_window"`
	TrustedProxies    []string      `yaml:"trusted_proxies"`
	AllowedOrigins    []string      `yaml:"allowed_origins"`
}

func Load() (*Config, error) {
	// Start with default configuration
	cfg := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Recaptcha: RecaptchaConfig{
			SecretKey:      "",
			VerifyURL:      "https://www.google.com/recaptcha/api/siteverify",
			ScoreThreshold: 0.5,
		},
		Email: EmailConfig{
			SMTPHost:     "",
			SMTPPort:     587,
			SMTPUsername: "",
			SMTPPassword: "",
			FromAddress:  "",
			ToAddress:    "",
			Subject:      "Nouveau message de contact",
			Format:       "html",
		},
		Security: SecurityConfig{
			MaxMessageSize:    4096,
			RateLimitRequests: 10,
			RateLimitWindow:   time.Minute,
			TrustedProxies:    nil,
		},
	}

	// Try to load from config file
	configPaths := []string{
		getEnv("GOSENDMAIL_CONFIG_FILE", ""),
		"/etc/gosendmail/config.yaml",
		"/usr/local/etc/gosendmail/config.yaml",
		"./config.yaml",
	}

	var configFile string
	var loaded bool

	for _, path := range configPaths {
		if path == "" {
			continue
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		if _, err := os.Stat(absPath); err == nil {
			if err := loadFromFile(absPath, cfg); err == nil {
				configFile = absPath
				loaded = true
				break
			}
		}
	}

	if !loaded {
		slog.Warn("No config file found, using defaults and environment variables")
	}

	// Override with environment variables (for sensitive data or runtime overrides)
	overrideWithEnv(cfg)

	// Validate final configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if loaded {
		slog.Info("Loaded configuration from file", "path", configFile)
	}

	return cfg, nil
}

func loadFromFile(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	return nil
}

func overrideWithEnv(cfg *Config) {
	// Server config
	if host := getEnv("SERVER_HOST", ""); host != "" {
		cfg.Server.Host = host
	}
	if port := getEnvInt("SERVER_PORT", 0); port != 0 {
		cfg.Server.Port = port
	}
	if timeout := getEnvDuration("SERVER_READ_TIMEOUT", 0); timeout != 0 {
		cfg.Server.ReadTimeout = timeout
	}
	if timeout := getEnvDuration("SERVER_WRITE_TIMEOUT", 0); timeout != 0 {
		cfg.Server.WriteTimeout = timeout
	}

	// Recaptcha config
	if secret := getEnv("RECAPTCHA_SECRET_KEY", ""); secret != "" {
		cfg.Recaptcha.SecretKey = secret
	}
	if url := getEnv("RECAPTCHA_VERIFY_URL", ""); url != "" {
		cfg.Recaptcha.VerifyURL = url
	}
	if threshold := getEnvFloat("RECAPTCHA_SCORE_THRESHOLD", 0); threshold != 0 {
		cfg.Recaptcha.ScoreThreshold = threshold
	}

	// Email config
	if host := getEnv("SMTP_HOST", ""); host != "" {
		cfg.Email.SMTPHost = host
	}
	if port := getEnvInt("SMTP_PORT", 0); port != 0 {
		cfg.Email.SMTPPort = port
	}
	if username := getEnv("SMTP_USERNAME", ""); username != "" {
		cfg.Email.SMTPUsername = username
	}
	if password := getEnv("SMTP_PASSWORD", ""); password != "" {
		cfg.Email.SMTPPassword = password
	}
	if from := getEnv("FROM_ADDRESS", ""); from != "" {
		cfg.Email.FromAddress = from
	}
	if to := getEnv("TO_ADDRESS", ""); to != "" {
		cfg.Email.ToAddress = to
	}
	if subject := getEnv("EMAIL_SUBJECT", ""); subject != "" {
		cfg.Email.Subject = subject
	}
	if format := getEnv("EMAIL_FORMAT", ""); format != "" {
		cfg.Email.Format = format
	}

	// Security config
	if size := getEnvInt("MAX_MESSAGE_SIZE", 0); size != 0 {
		cfg.Security.MaxMessageSize = size
	}
	if requests := getEnvInt("RATE_LIMIT_REQUESTS", 0); requests != 0 {
		cfg.Security.RateLimitRequests = requests
	}
	if window := getEnvDuration("RATE_LIMIT_WINDOW", 0); window != 0 {
		cfg.Security.RateLimitWindow = window
	}
	if proxies := getEnv("TRUSTED_PROXIES", ""); proxies != "" {
		cfg.Security.TrustedProxies = strings.Split(proxies, ",")
		for i := range cfg.Security.TrustedProxies {
			cfg.Security.TrustedProxies[i] = strings.TrimSpace(cfg.Security.TrustedProxies[i])
		}
	}
	if origins := getEnv("ALLOWED_ORIGINS", ""); origins != "" {
		cfg.Security.AllowedOrigins = strings.Split(origins, ",")
		for i := range cfg.Security.AllowedOrigins {
			cfg.Security.AllowedOrigins[i] = strings.TrimSpace(cfg.Security.AllowedOrigins[i])
		}
	}
}

func (c *Config) validate() error {
	if c.Recaptcha.SecretKey == "" {
		return fmt.Errorf("RECAPTCHA_SECRET_KEY is required")
	}
	if c.Email.SMTPHost == "" {
		return fmt.Errorf("SMTP_HOST is required")
	}
	if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	if c.Email.SMTPUsername == "" {
		return fmt.Errorf("SMTP_USERNAME is required")
	}
	if c.Email.SMTPPassword == "" {
		return fmt.Errorf("SMTP_PASSWORD is required")
	}
	if c.Email.FromAddress == "" {
		return fmt.Errorf("FROM_ADDRESS is required")
	}
	if c.Email.ToAddress == "" {
		return fmt.Errorf("TO_ADDRESS is required")
	}

	if _, err := mail.ParseAddress(c.Email.FromAddress); err != nil {
		return fmt.Errorf("invalid FROM_ADDRESS format")
	}
	if _, err := mail.ParseAddress(c.Email.ToAddress); err != nil {
		return fmt.Errorf("invalid TO_ADDRESS format")
	}

	if c.Email.Format != "html" && c.Email.Format != "text" {
		return fmt.Errorf("EMAIL_FORMAT must be 'html' or 'text'")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
