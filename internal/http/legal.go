package http

import (
	"html/template"
	"net/http"
)

type legalPageData struct {
	Title     string
	Eyebrow   string
	Body      template.HTML
	UpdatedOn string
}

func setLegalHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'none'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
	w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
}

func (s *server) renderLegalPage(w http.ResponseWriter, data legalPageData) {
	setLegalHeaders(w)
	_ = legalTpl.Execute(w, data)
}

func (s *server) handleLegalTerms(w http.ResponseWriter, r *http.Request) {
	s.renderLegalPage(w, legalPageData{
		Title:     "利用規約",
		Eyebrow:   "Legal",
		Body:      legalTermsBody,
		UpdatedOn: legalTermsUpdatedOn,
	})
}

func (s *server) handleLegalPrivacy(w http.ResponseWriter, r *http.Request) {
	s.renderLegalPage(w, legalPageData{
		Title:     "プライバシーポリシー",
		Eyebrow:   "Legal",
		Body:      legalPrivacyBody,
		UpdatedOn: legalPrivacyUpdatedOn,
	})
}

func (s *server) handleLegalContact(w http.ResponseWriter, r *http.Request) {
	s.renderLegalPage(w, legalPageData{
		Title:     "問い合わせ先",
		Eyebrow:   "Support",
		Body:      legalContactBody,
		UpdatedOn: legalContactUpdatedOn,
	})
}

func (s *server) handleLegalIncident(w http.ResponseWriter, r *http.Request) {
	s.renderLegalPage(w, legalPageData{
		Title:     "障害時連絡先",
		Eyebrow:   "Support",
		Body:      legalIncidentBody,
		UpdatedOn: legalIncidentUpdatedOn,
	})
}
