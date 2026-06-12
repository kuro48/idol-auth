package demo

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

type AuthorizationParams struct {
	HydraBrowserURL string
	ClientID        string
	RedirectURI     string
	State           string
	CodeChallenge   string
	Scopes          []string
}

func GeneratePKCEVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate pkce verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ComputeS256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func GenerateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func BuildAuthorizationURL(params AuthorizationParams) (string, error) {
	base := normalizeBaseURL(params.HydraBrowserURL)
	u, err := url.Parse(base + "/oauth2/auth")
	if err != nil {
		return "", fmt.Errorf("build authorization url: %w", err)
	}
	q := u.Query()
	q.Set("client_id", params.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", params.RedirectURI)
	q.Set("scope", strings.Join(params.Scopes, " "))
	q.Set("state", params.State)
	q.Set("code_challenge", params.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func normalizeBaseURL(raw string) string {
	return strings.TrimRight(raw, "/")
}
