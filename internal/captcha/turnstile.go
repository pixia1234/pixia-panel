package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TurnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func VerifyTurnstile(ctx context.Context, secret, token, remoteAddr string) (TurnstileResponse, error) {
	if secret == "" || token == "" {
		return TurnstileResponse{}, errors.New("missing secret or token")
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)

	if ip := extractIP(remoteAddr); ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return TurnstileResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return TurnstileResponse{}, err
	}
	defer resp.Body.Close()

	var parsed TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return TurnstileResponse{}, err
	}
	return parsed, nil
}

func extractIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}
