package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	turnstileVerifyURL     = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileResponseField = "cf-turnstile-response"
	turnstileVerifyTimeout = 10 * time.Second
	maxRegistrationBody    = 1 << 20
)

// TurnstileVerifier validates Cloudflare Turnstile tokens via the siteverify API.
type TurnstileVerifier struct {
	secretKey string
	endpoint  string
	client    *http.Client
}

func NewTurnstileVerifier(secretKey string) *TurnstileVerifier {
	return &TurnstileVerifier{
		secretKey: secretKey,
		endpoint:  turnstileVerifyURL,
		client:    &http.Client{Timeout: turnstileVerifyTimeout},
	}
}

func (v *TurnstileVerifier) Verify(ctx context.Context, token string) (bool, error) {
	form := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("turnstile: build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("turnstile: call siteverify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("turnstile: siteverify returned status %d", resp.StatusCode)
	}

	var decoded struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&decoded); err != nil {
		return false, fmt.Errorf("turnstile: decode siteverify response: %w", err)
	}
	return decoded.Success, nil
}

// ProtectRegistration wraps next and requires a valid Turnstile token on
// registration form submissions. A nil verifier disables the check entirely.
func ProtectRegistration(next http.Handler, verifier *TurnstileVerifier) http.Handler {
	if verifier == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/self-service/registration" {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxRegistrationBody))
		if err != nil {
			slog.ErrorContext(r.Context(), "turnstile: read registration body", "error", err)
			http.Error(w, "failed to read request", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))

		form, err := url.ParseQuery(string(body))
		if err != nil {
			slog.WarnContext(r.Context(), "turnstile: unparsable registration form")
			http.Redirect(w, r, "/registration", http.StatusSeeOther)
			return
		}

		token := strings.TrimSpace(form.Get(turnstileResponseField))
		if token == "" {
			slog.WarnContext(r.Context(), "turnstile: registration without token")
			http.Redirect(w, r, "/registration", http.StatusSeeOther)
			return
		}

		ok, err := verifier.Verify(r.Context(), token)
		if err != nil {
			slog.ErrorContext(r.Context(), "turnstile: verification failed", "error", err)
			http.Error(w, "bot verification temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if !ok {
			slog.WarnContext(r.Context(), "turnstile: token rejected")
			http.Redirect(w, r, "/registration", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
