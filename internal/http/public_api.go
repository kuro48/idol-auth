package http

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const maxPublicRequestBodyBytes = 1 << 16 // 64 KB

// handlePublicToken proxies POST /v1/public/api/token → Hydra /oauth2/token.
// The client sends standard OAuth2 form-encoded params (grant_type, code,
// client_id, client_secret, redirect_uri, code_verifier, etc.).
func (s *server) handlePublicToken(w http.ResponseWriter, r *http.Request) {
	body, ok := readLimitedBody(w, r, maxPublicRequestBodyBytes)
	if !ok {
		return
	}
	out, status, err := s.publicSvc.Token(r.Context(), body)
	if err != nil {
		slog.ErrorContext(r.Context(), "public token proxy error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

// handlePublicRevoke proxies POST /v1/public/api/token/revoke → Hydra /oauth2/revoke.
func (s *server) handlePublicRevoke(w http.ResponseWriter, r *http.Request) {
	body, ok := readLimitedBody(w, r, maxPublicRequestBodyBytes)
	if !ok {
		return
	}
	out, status, err := s.publicSvc.Revoke(r.Context(), body)
	if err != nil {
		slog.ErrorContext(r.Context(), "public revoke proxy error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

// handlePublicIntrospect proxies POST /v1/public/api/token/introspect → Hydra admin introspect.
func (s *server) handlePublicIntrospect(w http.ResponseWriter, r *http.Request) {
	body, ok := readLimitedBody(w, r, maxPublicRequestBodyBytes)
	if !ok {
		return
	}
	out, status, err := s.publicSvc.Introspect(r.Context(), body)
	if err != nil {
		slog.ErrorContext(r.Context(), "public introspect proxy error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

// handlePublicSession returns the Kratos session for the Bearer token in Authorization header.
func (s *server) handlePublicSession(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		http.Error(w, "Authorization: Bearer <token> required", http.StatusUnauthorized)
		return
	}
	session, err := s.publicSvc.GetSession(r.Context(), token)
	if err != nil {
		slog.ErrorContext(r.Context(), "public session lookup error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// handlePublicRegister handles POST /v1/public/api/register (headless account creation).
func (s *server) handlePublicRegister(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	result, err := s.publicSvc.Register(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "public register error", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "registration failed")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// handlePublicLogin handles POST /v1/public/api/login (headless password login).
func (s *server) handlePublicLogin(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	input.Identifier = strings.TrimSpace(input.Identifier)
	if input.Identifier == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}
	result, err := s.publicSvc.Login(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "public login error", "error", err)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// readLimitedBody reads the request body up to limit bytes.
// Returns false and writes an error response if reading fails.
func readLimitedBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return nil, false
	}
	return body, true
}
