package http

import (
	"net/http"

	"github.com/kuro48/idol-auth/internal/infra/webhook"
)

func (s *server) handlePatchAppWebhook(w http.ResponseWriter, r *http.Request) {
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		WebhookURL   string `json:"webhook_url"`
		RotateSecret bool   `json:"rotate_secret"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := webhook.ValidateURL(req.WebhookURL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, exists, err := s.webhookRepo.GetConfig(r.Context(), appActor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load webhook config")
		return
	}

	secret := cfg.WebhookSecret
	secretReturned := false
	if !exists || req.RotateSecret || secret == "" {
		generated, err := webhook.GenerateSecret()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate webhook secret")
			return
		}
		secret = generated
		secretReturned = true
	}

	if err := s.webhookRepo.SetConfig(r.Context(), appActor.ID, req.WebhookURL, secret); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save webhook config")
		return
	}

	resp := map[string]any{
		"webhook_url": req.WebhookURL,
	}
	if secretReturned {
		resp["webhook_secret"] = secret
	}
	writeJSON(w, http.StatusOK, resp)
}
