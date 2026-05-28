package http

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/kuro48/idol-auth/internal/domain/account"
)

type restorePageData struct {
	OshiColor    template.CSS
	DaysLeft     int
	ScheduledFor time.Time
	CSRFToken    string
	LogoutURL    string
}

func (s *server) handleRestoreAccount(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := s.authSvc.CurrentSession(r.Context(), r)
	if err != nil || !session.Authenticated || session.IdentityID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var req *account.DeletionRequest
	if s.accountSvc != nil {
		req, _ = s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	}
	if req == nil || req.Status != account.DeletionStatusScheduled {
		http.Redirect(w, r, "/account/", http.StatusSeeOther)
		return
	}

	nonce, err := newCSRFToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	oshiColor := template.CSS("#f472b6")
	if c := session.OshiColor; c != "" {
		oshiColor = template.CSS(c)
	}
	daysLeft := int(time.Until(req.ScheduledFor).Hours()/24) + 1
	if daysLeft < 0 {
		daysLeft = 0
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = restoreAccountTpl.Execute(w, restorePageData{
		OshiColor:    oshiColor,
		DaysLeft:     daysLeft,
		ScheduledFor: req.ScheduledFor,
		CSRFToken:    nonce,
		LogoutURL:    strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/logout",
	})
}

func (s *server) handleSubmitRestore(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := s.authSvc.CurrentSession(r.Context(), r)
	if err != nil || !session.Authenticated || session.IdentityID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if s.accountSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := s.accountSvc.CancelDeletion(r.Context(), session.IdentityID, session.IdentityID); err != nil {
		http.Error(w, "復活に失敗しました。もう一度お試しください。", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/account/", http.StatusSeeOther)
}
