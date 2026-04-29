package demo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

type stubSessionReader struct {
	session apphttp.KratosSession
	err     error
}

func (s *stubSessionReader) ToSession(_ context.Context, _ *http.Request) (apphttp.KratosSession, error) {
	if s.err != nil {
		return apphttp.KratosSession{}, s.err
	}
	return s.session, nil
}

type stubThemeUpdater struct {
	identityID string
	color      string
	err        error
}

func (s *stubThemeUpdater) SetIdentityOshiColor(_ context.Context, identityID, color string) error {
	s.identityID = identityID
	s.color = color
	return s.err
}

func TestResolveSessionOshiColorNormalizesStoredValue(t *testing.T) {
	reader := &stubSessionReader{
		session: apphttp.KratosSession{
			Active:    true,
			OshiColor: " #FFB2D8 ",
		},
	}

	got := ResolveSessionOshiColor(context.Background(), reader, httptest.NewRequest(http.MethodGet, "/", nil))
	if got != "#ffb2d8" {
		t.Fatalf("expected normalized color, got %q", got)
	}
}

func TestHandleThemePreferencePersistsNormalizedColor(t *testing.T) {
	reader := &stubSessionReader{
		session: apphttp.KratosSession{
			Active:     true,
			IdentityID: "identity-123",
		},
	}
	updater := &stubThemeUpdater{}
	req := httptest.NewRequest(http.MethodPost, "/ui/theme", strings.NewReader(`{"oshi_color":" #FFB2D8 "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleThemePreference(w, req, reader, updater)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if updater.identityID != "identity-123" {
		t.Fatalf("expected identity to be forwarded, got %q", updater.identityID)
	}
	if updater.color != "#ffb2d8" {
		t.Fatalf("expected normalized color to be persisted, got %q", updater.color)
	}
	if !strings.Contains(w.Body.String(), `"oshi_color":"#ffb2d8"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestHandleThemePreferenceRequiresAuthentication(t *testing.T) {
	reader := &stubSessionReader{err: apphttp.ErrNoActiveSession}
	req := httptest.NewRequest(http.MethodPost, "/ui/theme", strings.NewReader(`{"oshi_color":"#ffb2d8"}`))
	w := httptest.NewRecorder()

	HandleThemePreference(w, req, reader, &stubThemeUpdater{})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleThemePreferenceRejectsInvalidColor(t *testing.T) {
	reader := &stubSessionReader{
		session: apphttp.KratosSession{
			Active:     true,
			IdentityID: "identity-123",
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/ui/theme", strings.NewReader(`{"oshi_color":"#123456"}`))
	w := httptest.NewRecorder()

	HandleThemePreference(w, req, reader, &stubThemeUpdater{})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleThemePreferenceReturnsGatewayErrorOnPersistFailure(t *testing.T) {
	reader := &stubSessionReader{
		session: apphttp.KratosSession{
			Active:     true,
			IdentityID: "identity-123",
		},
	}
	updater := &stubThemeUpdater{err: errors.New("boom")}
	req := httptest.NewRequest(http.MethodPost, "/ui/theme", strings.NewReader(`{"oshi_color":"#ffb2d8"}`))
	w := httptest.NewRecorder()

	HandleThemePreference(w, req, reader, updater)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}
