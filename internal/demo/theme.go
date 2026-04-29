package demo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
	"github.com/ryunosukekurokawa/idol-auth/internal/oshi"
)

type SessionReader interface {
	ToSession(ctx context.Context, r *http.Request) (apphttp.KratosSession, error)
}

type ThemeUpdater interface {
	SetIdentityOshiColor(ctx context.Context, identityID, color string) error
}

func ResolveSessionOshiColor(ctx context.Context, reader SessionReader, r *http.Request) string {
	if reader == nil {
		return ""
	}
	session, err := reader.ToSession(ctx, r)
	if err != nil || !session.Active {
		return ""
	}
	return oshi.NormalizeColor(session.OshiColor)
}

func HandleThemePreference(w http.ResponseWriter, r *http.Request, reader SessionReader, updater ThemeUpdater) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if reader == nil || updater == nil {
		http.Error(w, "theme preference unavailable", http.StatusServiceUnavailable)
		return
	}

	session, err := reader.ToSession(r.Context(), r)
	if err != nil {
		if errors.Is(err, apphttp.ErrNoActiveSession) {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to resolve session", http.StatusBadGateway)
		return
	}
	if !session.Active || session.IdentityID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req struct {
		OshiColor string `json:"oshi_color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	color := oshi.NormalizeColor(req.OshiColor)
	if color == "" {
		http.Error(w, "invalid oshi color", http.StatusBadRequest)
		return
	}
	if err := updater.SetIdentityOshiColor(r.Context(), session.IdentityID, color); err != nil {
		http.Error(w, "failed to persist theme", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"oshi_color": color})
}
