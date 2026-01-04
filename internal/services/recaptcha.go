package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gosendmail/internal/config"
)

type RecaptchaService struct {
	secretKey      string
	verifyURL      string
	scoreThreshold float64
	client         *http.Client
}

type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
}

func NewRecaptchaService(cfg *config.Config) *RecaptchaService {
	return &RecaptchaService{
		secretKey:      cfg.Recaptcha.SecretKey,
		verifyURL:      cfg.Recaptcha.VerifyURL,
		scoreThreshold: cfg.Recaptcha.ScoreThreshold,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (rs *RecaptchaService) Verify(ctx context.Context, response, remoteIP string) error {
	if response == "" {
		return fmt.Errorf("reCAPTCHA response is empty")
	}

	data := url.Values{}
	data.Set("secret", rs.secretKey)
	data.Set("response", response)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rs.verifyURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create reCAPTCHA request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := rs.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to verify reCAPTCHA: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read reCAPTCHA response: %w", err)
	}

	var recaptchaResp RecaptchaResponse
	if err := json.Unmarshal(body, &recaptchaResp); err != nil {
		return fmt.Errorf("failed to parse reCAPTCHA response: %w", err)
	}

	if !recaptchaResp.Success {
		if len(recaptchaResp.ErrorCodes) > 0 {
			return fmt.Errorf("reCAPTCHA verification failed: %v", recaptchaResp.ErrorCodes)
		}
		return fmt.Errorf("reCAPTCHA verification failed")
	}

	// For reCAPTCHA v3, check the score
	if recaptchaResp.Score < rs.scoreThreshold {
		return fmt.Errorf("reCAPTCHA score %.2f is below threshold %.2f", recaptchaResp.Score, rs.scoreThreshold)
	}

	return nil
}
