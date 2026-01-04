# GoSendMail

A modern, secure Go backend for handling contact form submissions with reCAPTCHA v3 validation and email sending.

## Features

- Form field validation matching frontend regex patterns
- Google reCAPTCHA v3 validation with configurable score threshold
- SMTP email sending with TLS
- Rate limiting to prevent spam
- Graceful shutdown with signal handling
- Structured JSON logging
- Nonce-based CSRF protection
- CORS and security headers middleware
- Health and readiness endpoints
- Security best practices

## Quick Start

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Run the server:**
   ```bash
   go run main.go
   ```

## Configuration

The application uses environment variables and/or a YAML config file. See `.env.example` and `config.yaml.example` for all available options.

### Required Environment Variables

- `RECAPTCHA_SECRET_KEY` - Your Google reCAPTCHA v3 secret key
- `SMTP_HOST` - Your SMTP server hostname
- `SMTP_USERNAME` - SMTP username
- `SMTP_PASSWORD` - SMTP password (use app password for Gmail)
- `FROM_ADDRESS` - From email address
- `TO_ADDRESS` - Destination email address

### Optional Environment Variables

- `SERVER_HOST` - Server host (default: 0.0.0.0)
- `SERVER_PORT` - Server port (default: 8080)
- `RATE_LIMIT_REQUESTS` - Rate limit requests per window (default: 10)
- `RATE_LIMIT_WINDOW` - Rate limit time window (default: 1m)
- `RECAPTCHA_SCORE_THRESHOLD` - reCAPTCHA v3 score threshold (default: 0.5)
- `TRUSTED_PROXIES` - Comma-separated trusted proxy IPs (default: none)

## API Endpoints

| Method | Path            | Description                          |
|--------|-----------------|--------------------------------------|
| POST   | `/sendmail`     | Legacy HTML endpoint (returns HTML)  |
| POST   | `/contact`      | JSON REST endpoint (recommended)     |
| GET    | `/nonce`        | Generate a one-time CSRF nonce       |
| GET    | `/health`       | Health check                         |
| GET    | `/health/ready` | Readiness check (includes SMTP test) |

## Form Validation

The backend validates all form fields using the same regex patterns as your frontend:

- **firstName/name**: French letters, spaces, hyphens, and apostrophes, max 64 characters
- **postbox**: Numbers only, max 5 characters (optional)
- **street**: French letters, spaces, hyphens, apostrophes, max 128 characters
- **city**: French letters, spaces, hyphens, apostrophes, max 45 characters
- **zip**: Exactly 5 digits (French postal code format)
- **phone**: French phone format (0XXXXXXXXX)
- **email**: Standard email format (optional)
- **message**: Required, max 4096 characters

## Deployment

### Docker

```bash
docker build -t gosendmail .
docker run -p 8080:8080 --env-file .env gosendmail
```

### Debian Package

Download the latest `.deb` from [releases](https://github.com/robindubreuil/sendmail/releases) and install:

```bash
sudo dpkg -i gosendmail_*.deb
sudo systemctl edit gosendmail  # configure environment
sudo systemctl start gosendmail
```

### Systemd (manual)

```ini
[Unit]
Description=GoSendMail Service
After=network.target

[Service]
Type=simple
User=gosendmail
WorkingDirectory=/opt/gosendmail
ExecStart=/opt/gosendmail/gosendmail
Restart=always
RestartSec=5
EnvironmentFile=/opt/gosendmail/.env

[Install]
WantedBy=multi-user.target
```

## Testing

```bash
go test ./...
```

## Project Structure

```
main.go                    # Application entry point
internal/
  config/                  # Configuration management (YAML + env)
  handlers/                # HTTP request handlers
  middleware/              # CORS, rate limiting, logging, security headers
  services/                # Business logic (email, reCAPTCHA, nonce)
  models/                  # Data models and validation
  templates/               # Embedded email HTML templates
  util/                    # Shared utilities
debian/                    # Debian package configuration
docs/                      # Frontend form example
```

## License

MIT License
