package http

import (
	"net/http"
	"net/url"
	"strings"
)

// handlePublicBrowserLogin redirects the browser to the Hydra authorization endpoint.
// Query params forwarded: client_id, redirect_uri, scope, response_type, state,
// nonce, code_challenge, code_challenge_method.
func (s *server) handlePublicBrowserLogin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := map[string]string{
		"client_id":             q.Get("client_id"),
		"redirect_uri":          q.Get("redirect_uri"),
		"scope":                 q.Get("scope"),
		"response_type":         q.Get("response_type"),
		"state":                 q.Get("state"),
		"nonce":                 q.Get("nonce"),
		"code_challenge":        q.Get("code_challenge"),
		"code_challenge_method": q.Get("code_challenge_method"),
	}
	if params["client_id"] == "" || params["redirect_uri"] == "" || params["response_type"] == "" {
		writeError(w, http.StatusBadRequest, "client_id, redirect_uri and response_type are required")
		return
	}
	http.Redirect(w, r, s.publicSvc.LoginURL(params), http.StatusFound)
}

// handlePublicBrowserRegistration redirects the browser to the Kratos registration flow.
// Optional query param: return_to.
func (s *server) handlePublicBrowserRegistration(w http.ResponseWriter, r *http.Request) {
	returnTo := s.safePublicReturnTo(r.URL.Query().Get("return_to"))
	http.Redirect(w, r, s.publicSvc.RegistrationURL(returnTo), http.StatusFound) // #nosec G710 -- return_to is same-origin allowlisted before building the upstream URL.
}

// handlePublicBrowserLogout redirects the browser to the Hydra end-session endpoint.
// Optional query params: id_token_hint, post_logout_redirect_uri, state.
func (s *server) handlePublicBrowserLogout(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := map[string]string{
		"id_token_hint":            q.Get("id_token_hint"),
		"post_logout_redirect_uri": q.Get("post_logout_redirect_uri"),
		"state":                    q.Get("state"),
	}
	http.Redirect(w, r, s.publicSvc.LogoutURL(params), http.StatusFound)
}

func (s *server) safePublicReturnTo(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") {
		return strings.TrimRight(s.config.App.BaseURL, "/") + target
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return ""
	}
	if sameOrigin(u, s.config.App.BaseURL) {
		return target
	}
	return ""
}
