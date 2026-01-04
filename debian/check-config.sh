#!/bin/bash
# Configuration validation script for gosendmail service
# Executed before starting the service

set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

# Check required environment variables
check_required_env() {
    local missing=()

    if [[ -z "${RECAPTCHA_SECRET_KEY:-}" ]]; then
        missing+=("RECAPTCHA_SECRET_KEY")
    fi

    if [[ -z "${SMTP_HOST:-}" ]]; then
        missing+=("SMTP_HOST")
    fi

    if [[ -z "${SMTP_USERNAME:-}" ]]; then
        missing+=("SMTP_USERNAME")
    fi

    if [[ -z "${SMTP_PASSWORD:-}" ]]; then
        missing+=("SMTP_PASSWORD")
    fi

    if [[ -z "${FROM_ADDRESS:-}" ]]; then
        missing+=("FROM_ADDRESS")
    fi

    if [[ -z "${TO_ADDRESS:-}" ]]; then
        missing+=("TO_ADDRESS")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required environment variables:"
        for var in "${missing[@]}"; do
            log_error "  - $var"
        done
        log_error "Please configure these values in /etc/gosendmail/env"
        return 1
    fi
}

# Check email format
check_email_format() {
    local email="$1"
    if [[ ! "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        log_error "Invalid email format: $email"
        return 1
    fi
}

# Check if service can bind to configured port
check_port() {
    local port="${SERVER_PORT:-8080}"

    if [[ "$port" -lt 1 || "$port" -gt 65535 ]]; then
        log_error "Invalid port number: $port (must be 1-65535)"
        return 1
    fi

    # Check if port is already in use
    if command -v ss >/dev/null 2>&1; then
        if ss -tuln | grep -q ":$port\\s"; then
            log_warning "Port $port appears to be in use"
        fi
    fi
}

# Check directory permissions
check_directories() {
    local dirs=("/var/lib/gosendmail" "/var/log/gosendmail")

    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_error "Required directory does not exist: $dir"
            return 1
        fi

        if [[ ! -w "$dir" ]]; then
            log_error "Directory is not writable: $dir"
            return 1
        fi
    done
}

# Main validation
main() {
    log_info "Validating gosendmail configuration..."

    # Load environment files
    if [[ -f /etc/default/gosendmail ]]; then
        source /etc/default/gosendmail
    fi

    if [[ -f /etc/gosendmail/env ]]; then
        source /etc/gosendmail/env
    fi

    # Perform checks
    check_required_env
    check_email_format "$FROM_ADDRESS"
    check_email_format "$TO_ADDRESS"
    check_port
    check_directories

    log_info "Configuration validation passed"
    return 0
}

# Run validation
main "$@"