package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kuro48/idol-auth/internal/domain/profile"
	"github.com/kuro48/idol-auth/internal/oshi"
)

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
		if oshi.NormalizeColor(*req.OshiColor) == "" {
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
