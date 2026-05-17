package http_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	profiledomain "github.com/kuro48/idol-auth/internal/domain/profile"
	apphttp "github.com/kuro48/idol-auth/internal/http"
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
	cfg := testConfig()
	cfg.ProfileSvc = profileSvc
	return authenticatedProfileRouterWithConfig(cfg)
}

func authenticatedProfileRouterWithConfig(cfg apphttp.RouterConfig) http.Handler {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-1",
			Email:         "user@example.com",
		},
	}
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

func TestAccountCenter_RequiresAuthenticatedSession(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/account/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusSeeOther, w.Code, w.Body.String())
	}
	if got := w.Header().Get("Location"); !strings.Contains(got, "/self-service/login/browser") {
		t.Fatalf("expected login redirect, got %q", got)
	}
}

func TestAccountCenter_RendersAtBarePath(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-1",
			Email:         "user@example.com",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "アカウントセンター") {
		t.Fatalf("expected account center body, got %s", w.Body.String())
	}
}

func TestAccountCenter_RendersHTMLWhenAuthenticated(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-1",
			Email:         "user@example.com",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/account/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected html content type, got %q", ct)
	}
	body := w.Body.String()
	for _, fragment := range []string{
		"アカウントセンター",
		"共有プロフィール",
		"連携中アプリ",
		"アカウント削除",
		`name="avatar_file"`,
		`/v1/account/profile/avatar`,
		`name="locale"`,
		`name="timezone"`,
		`name="birthdate"`,
		`name="notify_email_enabled"`,
		`data-display="notification_preferences"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected body to contain %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "innerHTML = value") {
		t.Fatalf("expected profile display rendering to avoid raw innerHTML")
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
	avatarURL := "https://example.com/avatar.png"
	locale := "ja-JP"
	timezone := "Asia/Tokyo"
	birthdate := "2000-01-02"
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{
			IdentityID:  "identity-1",
			DisplayName: name,
			OshiColor:   color,
			AvatarURL:   avatarURL,
			Locale:      locale,
			Timezone:    timezone,
			Birthdate:   birthdate,
		},
	}
	router := authenticatedProfileRouter(svc)
	bodyJSON, _ := json.Marshal(map[string]any{
		"display_name": name,
		"oshi_color":   color,
		"avatar_url":   avatarURL,
		"locale":       locale,
		"timezone":     timezone,
		"birthdate":    birthdate,
		"notification_preferences": map[string]any{
			"email_enabled":                true,
			"product_updates":              false,
			"security_alerts":              true,
			"community_notifications":      true,
			"app_membership_notifications": true,
		},
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
	if svc.lastInput.AvatarURL == nil || *svc.lastInput.AvatarURL != avatarURL {
		t.Fatalf("expected avatar_url %q forwarded, got %v", avatarURL, svc.lastInput.AvatarURL)
	}
	if svc.lastInput.Locale == nil || *svc.lastInput.Locale != locale {
		t.Fatalf("expected locale %q forwarded, got %v", locale, svc.lastInput.Locale)
	}
	if svc.lastInput.Timezone == nil || *svc.lastInput.Timezone != timezone {
		t.Fatalf("expected timezone %q forwarded, got %v", timezone, svc.lastInput.Timezone)
	}
	if svc.lastInput.Birthdate == nil || *svc.lastInput.Birthdate != birthdate {
		t.Fatalf("expected birthdate %q forwarded, got %v", birthdate, svc.lastInput.Birthdate)
	}
	if svc.lastInput.NotificationPreferences == nil || !svc.lastInput.NotificationPreferences.SecurityAlerts {
		t.Fatalf("expected notification preferences forwarded, got %+v", svc.lastInput.NotificationPreferences)
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

func TestPatchProfile_AllowsEmptyOshiColor(t *testing.T) {
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{IdentityID: "identity-1", Locale: "ja-JP"},
	}
	router := authenticatedProfileRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"oshi_color":"","locale":"ja-JP"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if svc.lastInput.OshiColor == nil || *svc.lastInput.OshiColor != "" {
		t.Fatalf("expected empty oshi_color to be forwarded for clearing, got %v", svc.lastInput.OshiColor)
	}
	if svc.lastInput.Locale == nil || *svc.lastInput.Locale != "ja-JP" {
		t.Fatalf("expected locale update to be forwarded, got %v", svc.lastInput.Locale)
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

func TestPatchProfile_RejectsSelfManagedBadges(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"badges":[{"id":"top","label":"Top"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsInvalidAvatarURL(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBufferString(`{"avatar_url":"javascript:alert(1)"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUploadAvatar_StoresFileAndUpdatesProfile(t *testing.T) {
	pngData, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{IdentityID: "identity-1"},
	}
	cfg := testConfig()
	cfg.App.BaseURL = "https://auth.example.com"
	cfg.App.UploadDir = t.TempDir()
	cfg.ProfileSvc = svc
	router := authenticatedProfileRouterWithConfig(cfg)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(pngData); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/account/profile/avatar", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if svc.lastInput.AvatarURL == nil || !strings.HasPrefix(*svc.lastInput.AvatarURL, "https://auth.example.com/uploads/avatars/identity-1-") {
		t.Fatalf("expected generated avatar URL, got %v", svc.lastInput.AvatarURL)
	}
	filename := strings.TrimPrefix(*svc.lastInput.AvatarURL, "https://auth.example.com/uploads/avatars/")
	if _, err := os.Stat(cfg.App.UploadDir + "/avatars/" + filename); err != nil {
		t.Fatalf("expected avatar file to be stored: %v", err)
	}
}

func TestUploadAvatar_RejectsNonImage(t *testing.T) {
	cfg := testConfig()
	cfg.App.UploadDir = t.TempDir()
	cfg.ProfileSvc = &stubProfileService{}
	router := authenticatedProfileRouterWithConfig(cfg)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("avatar", "avatar.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("not an image")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/account/profile/avatar", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminPatchProfileAwards_UpdatesReadOnlyContributionFields(t *testing.T) {
	identityID := "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{
			IdentityID:          identityID,
			Badges:              []profiledomain.Badge{{ID: "top", Label: "Top Contributor"}},
			PrimaryBadgeID:      "top",
			ContributionScore:   42,
			ContributionSummary: profiledomain.ContributionSummary{PostsCount: 10},
		},
	}
	cfg := testConfig()
	cfg.ProfileSvc = svc
	router := apphttp.NewRouter(cfg, &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	body, _ := json.Marshal(map[string]any{
		"badges": []map[string]any{{
			"id":        "top",
			"label":     "Top Contributor",
			"level":     "gold",
			"issued_at": "2026-05-14T00:00:00Z",
		}},
		"primary_badge_id":   "top",
		"contribution_score": 42,
		"contribution_summary": map[string]any{
			"posts_count":         10,
			"helpful_votes_count": 100,
		},
	})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/"+identityID+"/profile-awards", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if svc.lastIdentityID != identityID {
		t.Fatalf("expected identity %q, got %q", identityID, svc.lastIdentityID)
	}
	if svc.lastInput.Badges == nil || len(*svc.lastInput.Badges) != 1 {
		t.Fatalf("expected badges forwarded, got %+v", svc.lastInput.Badges)
	}
	if svc.lastInput.ContributionScore == nil || *svc.lastInput.ContributionScore != 42 {
		t.Fatalf("expected contribution score forwarded, got %+v", svc.lastInput.ContributionScore)
	}
	if svc.lastInput.ContributionSummary == nil || svc.lastInput.ContributionSummary.PostsCount != 10 {
		t.Fatalf("expected contribution summary forwarded, got %+v", svc.lastInput.ContributionSummary)
	}
}

func TestAdminPatchProfileAwards_RejectsInvalidPrimaryBadge(t *testing.T) {
	identityID := "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"
	cfg := testConfig()
	cfg.ProfileSvc = &stubProfileService{}
	router := apphttp.NewRouter(cfg, &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/"+identityID+"/profile-awards", bytes.NewBufferString(`{"badges":[{"id":"top","label":"Top"}],"primary_badge_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}
