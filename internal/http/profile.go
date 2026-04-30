package http

import (
	"net/http"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
	"github.com/ryunosukekurokawa/idol-auth/internal/oshi"
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
		DisplayName *string   `json:"display_name"`
		OshiColor   *string   `json:"oshi_color"`
		OshiIDs     *[]string `json:"oshi_ids"`
		FanSince    *string   `json:"fan_since"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

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
		DisplayName: req.DisplayName,
		OshiColor:   req.OshiColor,
		OshiIDs:     req.OshiIDs,
		FanSince:    req.FanSince,
	}
	updated, err := s.profileSvc.UpdateProfile(r.Context(), session.IdentityID, input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
