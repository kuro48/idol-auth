package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	profiledomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

type stubProfileService struct {
	profile        profiledomain.Profile
	getErr         error
	updatedProfile profiledomain.Profile
	updateErr      error
	lastIdentityID string
	lastInput      profiledomain.UpdateInput
}

func (s *stubProfileService) GetProfile(_ context.Context, identityID string) (profiledomain.Profile, error) {
	s.lastIdentityID = identityID
	if s.getErr != nil {
		return profiledomain.Profile{}, s.getErr
	}
	return s.profile, nil
}

func (s *stubProfileService) UpdateProfile(_ context.Context, identityID string, input profiledomain.UpdateInput) (profiledomain.Profile, error) {
	s.lastIdentityID = identityID
	s.lastInput = input
	if s.updateErr != nil {
		return profiledomain.Profile{}, s.updateErr
	}
	if s.updatedProfile.IdentityID != "" {
		return s.updatedProfile, nil
	}
	return s.profile, nil
}

func authenticatedProfileRouter(profileSvc apphttp.ProfileService) http.Handler {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-1",
			Email:         "user@example.com",
		},
	}
	cfg := testConfig()
	cfg.ProfileSvc = profileSvc
	return apphttp.NewRouter(cfg, &stubAdminService{}, nil, authn, &stubAccountService{})
}

func TestGetProfile_RequiresAuthenticatedSession(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/account/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestGetProfile_ReturnsProfile_WhenAuthenticated(t *testing.T) {
	svc := &stubProfileService{
		profile: profiledomain.Profile{
			IdentityID:  "identity-1",
			DisplayName: "推し活太郎",
			OshiColor:   "#ffb2d8",
			OshiIDs:     []string{"idol-1"},
			FanSince:    "2019-04",
		},
	}
	router := authenticatedProfileRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/account/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"display_name"`) || !strings.Contains(body, "推し活太郎") {
		t.Fatalf("expected profile in response body, got %s", body)
	}
	if svc.lastIdentityID != "identity-1" {
		t.Fatalf("expected identity id forwarded to service, got %q", svc.lastIdentityID)
	}
}

func TestGetProfile_ReturnsServiceUnavailable_WhenNoProfileSvc(t *testing.T) {
	router := authenticatedProfileRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/account/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusServiceUnavailable, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RequiresAuthenticatedSession(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"display_name":"テスト"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestPatchProfile_UpdatesAndReturnsProfile(t *testing.T) {
	name := "新しい名前"
	color := "#ffb2d8"
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{
			IdentityID:  "identity-1",
			DisplayName: name,
			OshiColor:   color,
		},
	}
	router := authenticatedProfileRouter(svc)
	bodyJSON, _ := json.Marshal(map[string]any{
		"display_name": name,
		"oshi_color":   color,
	})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if svc.lastIdentityID != "identity-1" {
		t.Fatalf("expected identity id forwarded to service, got %q", svc.lastIdentityID)
	}
	if svc.lastInput.DisplayName == nil || *svc.lastInput.DisplayName != name {
		t.Fatalf("expected display_name %q forwarded, got %v", name, svc.lastInput.DisplayName)
	}
	if svc.lastInput.OshiColor == nil || *svc.lastInput.OshiColor != color {
		t.Fatalf("expected oshi_color %q forwarded, got %v", color, svc.lastInput.OshiColor)
	}
}

func TestPatchProfile_RejectsBadJSON(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsEmptyDisplayName(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"display_name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsInvalidOshiColor(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"oshi_color":"#123456"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsFutureFanSince(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"fan_since":"9999-12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsEmptyOshiIDElement(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	body, _ := json.Marshal(map[string]any{"oshi_ids": []string{""}})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}
