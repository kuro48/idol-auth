package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
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
		"推し活プロフィール",
		"連携中の推し活サービス",
		"アカウント削除",
		`name="avatar_file"`,
		`id="avatar-file"`,
		`id="avatar-editor"`,
		`id="avatar-crop-x"`,
		`id="avatar-crop-y"`,
		`id="avatar-zoom"`,
		`id="avatar-rotate-left"`,
		`id="avatar-rotate-right"`,
		`id="avatar-canvas"`,
		`--avatar-display-size:146px`,
		`--avatar-display-radius:36px`,
		`width:var(--avatar-display-size); height:var(--avatar-display-size); border-radius:var(--avatar-display-radius)`,
		`border-radius:var(--avatar-display-radius)`,
		`.profile-photo.has-image{color:transparent; background-size:cover; background-position:center; background-repeat:no-repeat}`,
		`function renderAvatarEditor()`,
		`function exportEditedAvatarBlob()`,
		`var AVATAR_MAX_UPLOAD_BYTES = 10 * 1024 * 1024`,
		`function assertAvatarSize(blob)`,
		`function avatarFileInput(form)`,
		`querySelector('input[name="avatar_file"]')`,
		`画像サイズが10MBを超えています`,
		`/v1/account/profile/avatar`,
		`name="locale"`,
		`name="timezone"`,
		`name="birthdate"`,
		`name="notify_email_enabled"`,
		`data-display="notification_preferences"`,
		`function showError(err)`,
		`showToast(err && err.message ? err.message : 'エラーが発生しました', 'error')`,
		`.toast.error`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected body to contain %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "innerHTML = value") {
		t.Fatalf("expected profile display rendering to avoid raw innerHTML")
	}
	if strings.Contains(body, "form.avatar_file") || strings.Contains(body, "this.avatar_file") {
		t.Fatalf("expected avatar file input access to use explicit selector")
	}
}

func TestGetProfile_ReturnsProfile_WhenAuthenticated(t *testing.T) {
	svc := &stubProfileService{
		profile: profiledomain.Profile{
			IdentityID:  "identity-1",
			DisplayName: "推し活太郎",
			OshiColor:   "#ffb2d8",
			Oshis:       []profiledomain.OshiEntry{{IdolID: "idol-1", FanSince: "2019-04"}},
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

func TestPatchProfile_RejectsOshiWithFutureFanSince(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	body, _ := json.Marshal(map[string]any{
		"oshis": []map[string]any{{"idol_id": "idol-1", "fan_since": "9999-12"}},
	})
	req := httptest.NewRequest(http.MethodPatch, "/v1/account/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPatchProfile_RejectsOshiWithEmptyIdolID(t *testing.T) {
	router := authenticatedProfileRouter(&stubProfileService{})
	body, _ := json.Marshal(map[string]any{
		"oshis": []map[string]any{{"idol_id": ""}},
	})
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
	pngData := testPNG(t)
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
	if !strings.HasSuffix(*svc.lastInput.AvatarURL, ".jpg") {
		t.Fatalf("expected generated avatar URL to use normalized jpeg extension, got %q", *svc.lastInput.AvatarURL)
	}
	filename := strings.TrimPrefix(*svc.lastInput.AvatarURL, "https://auth.example.com/uploads/avatars/")
	storedPath := cfg.App.UploadDir + "/avatars/" + filename
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("expected avatar file to be stored: %v", err)
	}
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored avatar: %v", err)
	}
	if got := http.DetectContentType(stored); got != "image/jpeg" {
		t.Fatalf("expected stored avatar to be image/jpeg, got %q", got)
	}
}

func TestUploadAvatar_AcceptsPhotoOverTwoMBAndNormalizes(t *testing.T) {
	jpegData := testLargeJPEG(t)
	if len(jpegData) <= 2<<20 {
		t.Fatalf("expected fixture to exceed old 2MB limit, got %d bytes", len(jpegData))
	}
	if len(jpegData) >= 10<<20 {
		t.Fatalf("expected fixture to stay below 10MB limit, got %d bytes", len(jpegData))
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
	part, err := mw.CreateFormFile("avatar", "photo.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(jpegData); err != nil {
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
	filename := strings.TrimPrefix(*svc.lastInput.AvatarURL, "https://auth.example.com/uploads/avatars/")
	stored, err := os.ReadFile(cfg.App.UploadDir + "/avatars/" + filename)
	if err != nil {
		t.Fatalf("read stored avatar: %v", err)
	}
	if len(stored) >= len(jpegData) {
		t.Fatalf("expected normalized avatar to be smaller than original, original=%d stored=%d", len(jpegData), len(stored))
	}
	cfgImg, _, err := image.DecodeConfig(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("decode stored avatar config: %v", err)
	}
	if cfgImg.Width != 512 || cfgImg.Height != 512 {
		t.Fatalf("expected stored avatar to be 512x512, got %dx%d", cfgImg.Width, cfgImg.Height)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 10), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}
	return buf.Bytes()
}

func testLargeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1800, 1800))
	var seed uint32 = 1
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			seed = seed*1664525 + 1013904223
			img.Set(x, y, color.RGBA{
				R: uint8(seed >> 24),
				G: uint8(seed >> 16),
				B: uint8(seed >> 8),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	return buf.Bytes()
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

func TestAdminPatchProfileAwards_UpdatesBadges(t *testing.T) {
	identityID := "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"
	svc := &stubProfileService{
		updatedProfile: profiledomain.Profile{
			IdentityID:     identityID,
			Badges:         []profiledomain.Badge{{ID: "top", Label: "Top Contributor"}},
			PrimaryBadgeID: "top",
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
		"primary_badge_id": "top",
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
	if svc.lastInput.PrimaryBadgeID == nil || *svc.lastInput.PrimaryBadgeID != "top" {
		t.Fatalf("expected primary_badge_id forwarded, got %+v", svc.lastInput.PrimaryBadgeID)
	}
}

func TestGetPublicUserProfile_RequiresAuthentication(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/users/other-user/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestGetPublicUserProfile_ReturnsPublicViewWithoutPII(t *testing.T) {
	svc := &stubProfileService{
		profile: profiledomain.Profile{
			IdentityID:  "other-user",
			DisplayName: "他のユーザー",
			OshiColor:   "#b2ffb2",
			Oshis:       []profiledomain.OshiEntry{{IdolID: "idol-2", FanSince: "2021"}},
			Email:       "other@example.com",
		},
	}
	cfg := testConfig()
	cfg.ProfileSvc = svc
	router := authenticatedProfileRouterWithConfig(cfg)
	req := httptest.NewRequest(http.MethodGet, "/v1/users/other-user/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "other@example.com") {
		t.Fatalf("response must not contain email PII, got %s", body)
	}
	if !strings.Contains(body, "他のユーザー") {
		t.Fatalf("expected display_name in response, got %s", body)
	}
	if !strings.Contains(body, "idol-2") {
		t.Fatalf("expected oshi data in response, got %s", body)
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
