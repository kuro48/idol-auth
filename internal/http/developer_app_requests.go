package http

import (
	"context"
	"crypto/subtle"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

const developerCSRFCookieName = "idol_auth_devreq_csrf"

// DeveloperAppRegService is the subset of appreg.Service used by developer-facing handlers.
type DeveloperAppRegService interface {
	Submit(ctx context.Context, ownerID string, input appreg.SubmitInput) (appreg.Request, error)
	GetForOwner(ctx context.Context, id uuid.UUID, ownerID string) (appreg.Request, error)
	ListMine(ctx context.Context, ownerID string) ([]appreg.Request, error)
	Withdraw(ctx context.Context, id uuid.UUID, ownerID string) (appreg.Request, error)
	Resubmit(ctx context.Context, id uuid.UUID, ownerID string, input appreg.SubmitInput) (appreg.Request, error)
}

// devPageBase holds fields shared by all developer HTML page templates.
type devPageBase struct {
	Initials  string
	Email     string
	OshiColor template.CSS
	LogoutURL string
}

type developerRequestListData struct {
	devPageBase
	Requests []appreg.Request
}

type developerRequestFormData struct {
	devPageBase
	CSRFToken string
	Error     string
	Req       *appreg.Request // non-nil when resubmitting
}

type developerRequestDetailData struct {
	devPageBase
	CSRFToken string
	Req       *appreg.Request
}

// --- CSRF helpers ---

func setDeveloperCSRFCookie(w http.ResponseWriter, token string, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     developerCSRFCookieName,
		Value:    token,
		Path:     "/developer/app-requests",
		Domain:   domain,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   1800,
	})
}

func validateDeveloperCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(developerCSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := strings.TrimSpace(r.FormValue("csrf_token"))
	return formToken != "" && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
}

func (s *server) newDeveloperCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	token, err := newCSRFToken()
	if err != nil {
		return "", err
	}
	secure := s.config.Security.CookieSecure && requestIsSecure(r, s.config.Security.TrustedProxies)
	setDeveloperCSRFCookie(w, token, secure, s.config.Security.CookieDomain)
	return token, nil
}

// --- Helpers ---

// splitLines splits a textarea value by newlines, trims whitespace, and drops blanks.
func splitLines(raw string) []string {
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func devPageBaseFromSession(s *server, session SessionView) devPageBase {
	oshiColor := template.CSS("#1740c9")
	if c := session.OshiColor; c != "" {
		oshiColor = template.CSS(c)
	}
	return devPageBase{
		Initials:  initials(session.DisplayName, session.Email),
		Email:     session.Email,
		OshiColor: oshiColor,
		LogoutURL: strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/logout",
	}
}

func parseSubmitInputFromForm(r *http.Request) appreg.SubmitInput {
	return appreg.SubmitInput{
		Name:                   strings.TrimSpace(r.FormValue("name")),
		Type:                   strings.TrimSpace(r.FormValue("type")),
		Description:            strings.TrimSpace(r.FormValue("description")),
		Purpose:                strings.TrimSpace(r.FormValue("purpose")),
		ContactEmail:           strings.TrimSpace(r.FormValue("contact_email")),
		HomepageURL:            strings.TrimSpace(r.FormValue("homepage_url")),
		PrivacyPolicyURL:       strings.TrimSpace(r.FormValue("privacy_policy_url")),
		TermsURL:               strings.TrimSpace(r.FormValue("terms_url")),
		Organization:           strings.TrimSpace(r.FormValue("organization")),
		RedirectURIs:           splitLines(r.FormValue("redirect_uris")),
		PostLogoutRedirectURIs: splitLines(r.FormValue("post_logout_redirect_uris")),
		Scopes:                 splitLines(r.FormValue("scopes")),
	}
}

func friendlyAppRegError(err error) string {
	switch {
	case errors.Is(err, appreg.ErrInvalidName):
		return "アプリ名を確認してください（1〜100文字）"
	case errors.Is(err, appreg.ErrInvalidType):
		return "アプリ種別を選択してください（web / spa / native / m2m）"
	case errors.Is(err, appreg.ErrInvalidDescription):
		return "説明を入力してください（1〜1000文字）"
	case errors.Is(err, appreg.ErrInvalidPurpose):
		return "利用目的を入力してください（200〜2000文字）"
	case errors.Is(err, appreg.ErrInvalidEmail):
		return "連絡先メールアドレスの形式が正しくありません"
	case errors.Is(err, appreg.ErrInvalidRedirectURI):
		return "リダイレクトURIの形式が正しくありません"
	case errors.Is(err, appreg.ErrInvalidURL):
		return "URLの形式が正しくありません（https必須）"
	case errors.Is(err, appreg.ErrRedirectURIRequired):
		return "リダイレクトURIは必須です（m2m以外）"
	case errors.Is(err, appreg.ErrCannotResubmit):
		return "現在の状態では再申請できません"
	case errors.Is(err, appreg.ErrCannotWithdraw):
		return "現在の状態では取り下げできません"
	case errors.Is(err, appreg.ErrDuplicatePending):
		return "未完了の申請がすでに存在します"
	default:
		return "エラーが発生しました。しばらく後にもう一度お試しください"
	}
}

// --- HTML handlers ---

func (s *server) handleDeveloperAppRequestsList(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	reqs, err := s.developerAppRegSvc.ListMine(r.Context(), session.IdentityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list developer app requests", "error", err)
		http.Error(w, "error loading requests", http.StatusInternalServerError)
		return
	}
	if reqs == nil {
		reqs = []appreg.Request{}
	}
	setAccountCenterHeaders(w)
	_ = developerRequestListTpl.Execute(w, developerRequestListData{
		devPageBase: devPageBaseFromSession(s, session),
		Requests:    reqs,
	})
}

func (s *server) handleDeveloperAppRequestsNew(w http.ResponseWriter, r *http.Request) {
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	csrfToken, err := s.newDeveloperCSRFToken(w, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setAccountCenterHeaders(w)
	_ = developerRequestFormTpl.Execute(w, developerRequestFormData{
		devPageBase: devPageBaseFromSession(s, session),
		CSRFToken:   csrfToken,
	})
}

func (s *server) handleDeveloperAppRequestsCreate(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !validateDeveloperCSRF(r) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	input := parseSubmitInputFromForm(r)
	req, err := s.developerAppRegSvc.Submit(r.Context(), session.IdentityID, input)
	if err != nil {
		csrfToken, _ := s.newDeveloperCSRFToken(w, r)
		setAccountCenterHeaders(w)
		_ = developerRequestFormTpl.Execute(w, developerRequestFormData{
			devPageBase: devPageBaseFromSession(s, session),
			CSRFToken:   csrfToken,
			Error:       friendlyAppRegError(err),
		})
		return
	}
	http.Redirect(w, r, "/developer/app-requests/"+req.ID.String(), http.StatusSeeOther)
}

func (s *server) handleDeveloperAppRequestDetail(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.GetForOwner(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	csrfToken, err := s.newDeveloperCSRFToken(w, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setAccountCenterHeaders(w)
	_ = developerRequestDetailTpl.Execute(w, developerRequestDetailData{
		devPageBase: devPageBaseFromSession(s, session),
		CSRFToken:   csrfToken,
		Req:         &req,
	})
}

func (s *server) handleDeveloperAppRequestEdit(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.GetForOwner(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	csrfToken, err := s.newDeveloperCSRFToken(w, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setAccountCenterHeaders(w)
	_ = developerRequestFormTpl.Execute(w, developerRequestFormData{
		devPageBase: devPageBaseFromSession(s, session),
		CSRFToken:   csrfToken,
		Req:         &req,
	})
}

func (s *server) handleDeveloperAppRequestResubmit(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !validateDeveloperCSRF(r) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	input := parseSubmitInputFromForm(r)
	req, err := s.developerAppRegSvc.Resubmit(r.Context(), id, session.IdentityID, input)
	if err != nil {
		orig, _ := s.developerAppRegSvc.GetForOwner(r.Context(), id, session.IdentityID)
		csrfToken, _ := s.newDeveloperCSRFToken(w, r)
		setAccountCenterHeaders(w)
		_ = developerRequestFormTpl.Execute(w, developerRequestFormData{
			devPageBase: devPageBaseFromSession(s, session),
			CSRFToken:   csrfToken,
			Error:       friendlyAppRegError(err),
			Req:         &orig,
		})
		return
	}
	http.Redirect(w, r, "/developer/app-requests/"+req.ID.String(), http.StatusSeeOther)
}

func (s *server) handleDeveloperAppRequestWithdraw(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !validateDeveloperCSRF(r) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	if _, err := s.developerAppRegSvc.Withdraw(r.Context(), id, session.IdentityID); err != nil {
		writeAppRegError(w, err)
		return
	}
	http.Redirect(w, r, "/developer/app-requests", http.StatusSeeOther)
}

// --- JSON API handlers ---

func (s *server) handleListMyAppRequests(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	reqs, err := s.developerAppRegSvc.ListMine(r.Context(), session.IdentityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list my app requests", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if reqs == nil {
		reqs = []appreg.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reqs})
}

func (s *server) handleSubmitAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input appreg.SubmitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req, err := s.developerAppRegSvc.Submit(r.Context(), session.IdentityID, input)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (s *server) handleGetMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.GetForOwner(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) handleWithdrawMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.Withdraw(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) handleResubmitMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var input appreg.SubmitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req, err := s.developerAppRegSvc.Resubmit(r.Context(), id, session.IdentityID, input)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}
