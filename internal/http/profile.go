package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kuro48/idol-auth/internal/domain/profile"
	"github.com/kuro48/idol-auth/internal/oshi"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxAvatarUploadBytes int64 = 10 << 20
const maxAvatarRequestBytes int64 = maxAvatarUploadBytes + 512<<10
const avatarOutputSize = 512
const maxAvatarDimension = 12000
const maxAvatarPixels = 50_000_000

func (s *server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "profile service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	p, err := s.profileSvc.GetProfile(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) handlePatchProfile(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "profile service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		DisplayName             *string                          `json:"display_name"`
		AvatarURL               *string                          `json:"avatar_url"`
		Locale                  *string                          `json:"locale"`
		Timezone                *string                          `json:"timezone"`
		Birthdate               *string                          `json:"birthdate"`
		NotificationPreferences *profile.NotificationPreferences `json:"notification_preferences"`
		OshiColor               *string                          `json:"oshi_color"`
		OshiIDs                 *[]string                        `json:"oshi_ids"`
		FanSince                *string                          `json:"fan_since"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	trimStringPtr(req.DisplayName)
	trimStringPtr(req.AvatarURL)
	trimStringPtr(req.Locale)
	trimStringPtr(req.Timezone)
	trimStringPtr(req.Birthdate)
	trimStringPtr(req.OshiColor)
	trimStringPtr(req.FanSince)

	if req.DisplayName != nil {
		if err := profile.ValidateDisplayName(*req.DisplayName); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.OshiColor != nil {
		if *req.OshiColor != "" && oshi.NormalizeColor(*req.OshiColor) == "" {
			writeError(w, http.StatusBadRequest, "invalid oshi_color")
			return
		}
	}
	if req.AvatarURL != nil {
		if err := profile.ValidateAvatarURL(*req.AvatarURL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Locale != nil {
		if err := profile.ValidateLocale(*req.Locale); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Timezone != nil {
		if err := profile.ValidateTimezone(*req.Timezone); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Birthdate != nil {
		if err := profile.ValidateBirthdate(*req.Birthdate, time.Now().UTC()); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.FanSince != nil {
		if err := profile.ValidateFanSince(*req.FanSince, time.Now().UTC()); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.OshiIDs != nil {
		if err := profile.ValidateOshiIDs(*req.OshiIDs); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	input := profile.UpdateInput{
		DisplayName:             req.DisplayName,
		AvatarURL:               req.AvatarURL,
		Locale:                  req.Locale,
		Timezone:                req.Timezone,
		Birthdate:               req.Birthdate,
		NotificationPreferences: req.NotificationPreferences,
		OshiColor:               req.OshiColor,
		OshiIDs:                 req.OshiIDs,
		FanSince:                req.FanSince,
	}
	updated, err := s.profileSvc.UpdateProfile(r.Context(), session.IdentityID, input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *server) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "profile service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarRequestBytes)
	if err := r.ParseMultipartForm(maxAvatarUploadBytes); err != nil { // #nosec G120 -- body is capped by MaxBytesReader before multipart parsing.
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read avatar file")
		return
	}
	if int64(len(data)) > maxAvatarUploadBytes {
		writeError(w, http.StatusBadRequest, "avatar file is too large")
		return
	}
	if !isSupportedAvatarInput(data) {
		writeError(w, http.StatusBadRequest, "avatar must be a png, jpeg, gif, or webp image")
		return
	}
	avatarData, err := normalizeAvatarImage(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sum := sha256.Sum256(avatarData)
	filename := sanitizeAvatarOwner(session.IdentityID) + "-" + hex.EncodeToString(sum[:])[:16] + ".jpg"
	dir := filepath.Join(s.config.App.UploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare avatar storage")
		return
	}
	if err := os.WriteFile(filepath.Join(dir, filename), avatarData, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store avatar file")
		return
	}

	avatarURL := strings.TrimRight(s.config.App.BaseURL, "/") + "/uploads/avatars/" + url.PathEscape(filename)
	if err := profile.ValidateAvatarURL(avatarURL); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid generated avatar url")
		return
	}
	updated, err := s.profileSvc.UpdateProfile(r.Context(), session.IdentityID, profile.UpdateInput{
		AvatarURL: &avatarURL,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *server) handleAvatarAsset(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "file")
	if filename == "" || filename != filepath.Base(filename) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.config.App.UploadDir, "avatars", filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, path)
}

func (s *server) handlePatchProfileAwards(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "profile service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
		return
	}

	var req struct {
		Badges              *[]profile.Badge             `json:"badges"`
		PrimaryBadgeID      *string                      `json:"primary_badge_id"`
		ContributionScore   *int                         `json:"contribution_score"`
		ContributionSummary *profile.ContributionSummary `json:"contribution_summary"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	trimStringPtr(req.PrimaryBadgeID)
	if req.Badges == nil && req.PrimaryBadgeID == nil && req.ContributionScore == nil && req.ContributionSummary == nil {
		writeError(w, http.StatusBadRequest, "at least one award field is required")
		return
	}
	var badges []profile.Badge
	if req.Badges != nil {
		badges = *req.Badges
		if err := profile.ValidateBadges(badges); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.PrimaryBadgeID != nil {
		if req.Badges == nil {
			current, err := s.profileSvc.GetProfile(r.Context(), identityID)
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to load profile")
				return
			}
			badges = current.Badges
		}
		if err := profile.ValidatePrimaryBadgeID(*req.PrimaryBadgeID, badges); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.ContributionScore != nil {
		if err := profile.ValidateContributionScore(*req.ContributionScore); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.ContributionSummary != nil {
		if err := profile.ValidateContributionSummary(*req.ContributionSummary); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	updated, err := s.profileSvc.UpdateProfile(r.Context(), identityID, profile.UpdateInput{
		Badges:              req.Badges,
		PrimaryBadgeID:      req.PrimaryBadgeID,
		ContributionScore:   req.ContributionScore,
		ContributionSummary: req.ContributionSummary,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update profile awards")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func trimStringPtr(s *string) {
	if s != nil {
		*s = strings.TrimSpace(*s)
	}
}

func isSupportedAvatarInput(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return true
	}
	return false
}

func normalizeAvatarImage(data []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("avatar image could not be decoded")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errors.New("avatar image has invalid dimensions")
	}
	if cfg.Width > maxAvatarDimension || cfg.Height > maxAvatarDimension || cfg.Width*cfg.Height > maxAvatarPixels {
		return nil, errors.New("avatar image dimensions are too large")
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("avatar image could not be decoded")
	}
	srcRect := centerSquareBounds(src.Bounds())
	dst := image.NewRGBA(image.Rect(0, 0, avatarOutputSize, avatarOutputSize))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, srcRect, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, errors.New("avatar image could not be encoded")
	}
	return out.Bytes(), nil
}

func centerSquareBounds(b image.Rectangle) image.Rectangle {
	w := b.Dx()
	h := b.Dy()
	size := w
	if h < size {
		size = h
	}
	x0 := b.Min.X + (w-size)/2
	y0 := b.Min.Y + (h-size)/2
	return image.Rect(x0, y0, x0+size, y0+size)
}

func sanitizeAvatarOwner(identityID string) string {
	var b strings.Builder
	for _, r := range identityID {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "avatar"
	}
	return b.String()
}
