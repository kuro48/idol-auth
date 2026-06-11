package appreg

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
)

var (
	ErrInvalidName         = errors.New("name must be 1-100 characters")
	ErrInvalidType         = errors.New("type must be one of: web, spa, native, m2m")
	ErrInvalidDescription  = errors.New("description must be 1-1000 characters")
	ErrInvalidPurpose      = errors.New("purpose must be 200-2000 characters")
	ErrScopeNotAllowed     = errors.New("scope is not allowed for self-service registration")
	ErrInvalidEmail        = errors.New("contact_email is not a valid email address")
	ErrInvalidRedirectURI  = errors.New("redirect_uri is not a valid absolute https or http://localhost URI")
	ErrInvalidURL          = errors.New("url is not a valid absolute https URI")
	ErrRedirectURIRequired = errors.New("redirect_uris is required for this app type")
)

var validTypes = map[string]bool{
	"web": true, "spa": true, "native": true, "m2m": true,
}

// selfServiceAllowedScopes limits instantly issued clients to standard OIDC scopes.
var selfServiceAllowedScopes = map[string]bool{
	"openid": true, "email": true, "profile": true, "offline_access": true,
}

// SubmitInput holds validated fields for a new registration request.
type SubmitInput struct {
	Name                   string
	Type                   string
	Description            string
	HomepageURL            string
	PrivacyPolicyURL       string
	TermsURL               string
	ContactEmail           string
	Organization           string
	Purpose                string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	Scopes                 []string
	// SelfService relaxes the purpose requirement and restricts scopes to the
	// allowlist, for instantly issued (auto-approved) registrations.
	SelfService bool
}

// Validate checks all fields and normalizes string values in-place.
func (in *SubmitInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) == 0 || len(in.Name) > 100 {
		return ErrInvalidName
	}

	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	if !validTypes[in.Type] {
		return ErrInvalidType
	}

	in.Description = strings.TrimSpace(in.Description)
	if len(in.Description) == 0 || len(in.Description) > 1000 {
		return ErrInvalidDescription
	}

	in.Purpose = strings.TrimSpace(in.Purpose)
	purposeMin := 200
	if in.SelfService {
		purposeMin = 0
	}
	if len(in.Purpose) < purposeMin || len(in.Purpose) > 2000 {
		return ErrInvalidPurpose
	}

	in.ContactEmail = strings.TrimSpace(in.ContactEmail)
	if _, err := mail.ParseAddress(in.ContactEmail); err != nil {
		return ErrInvalidEmail
	}

	for _, u := range []*string{&in.HomepageURL, &in.PrivacyPolicyURL, &in.TermsURL} {
		*u = strings.TrimSpace(*u)
		if err := validateOptionalHTTPSURL(*u); err != nil {
			return err
		}
	}

	uris, err := normalizeRedirectURIs(in.RedirectURIs, in.Type)
	if err != nil {
		return err
	}
	if in.Type != "m2m" && len(uris) == 0 {
		return ErrRedirectURIRequired
	}
	in.RedirectURIs = uris

	postLogout, err := normalizeRedirectURIs(in.PostLogoutRedirectURIs, in.Type)
	if err != nil {
		return err
	}
	in.PostLogoutRedirectURIs = postLogout
	in.Scopes = normalizeScopes(in.Scopes)
	if in.SelfService {
		for _, scope := range in.Scopes {
			if !selfServiceAllowedScopes[scope] {
				return ErrScopeNotAllowed
			}
		}
	}
	in.Organization = strings.TrimSpace(in.Organization)

	return nil
}

func validateOptionalHTTPSURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func normalizeRedirectURIs(uris []string, appType string) ([]string, error) {
	seen := make(map[string]bool, len(uris))
	out := make([]string, 0, len(uris))
	for _, raw := range uris {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		if err := validateRedirectURI(raw, appType); err != nil {
			return nil, err
		}
		seen[raw] = true
		out = append(out, raw)
	}
	return out, nil
}

func validateRedirectURI(raw, appType string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrInvalidRedirectURI
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && isLoopback(u.Hostname()) {
		return nil
	}
	// custom URI schemes allowed for native apps (e.g. myapp://)
	if appType == "native" && u.Scheme != "" && u.Scheme != "http" && u.Scheme != "javascript" {
		return nil
	}
	return ErrInvalidRedirectURI
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func normalizeScopes(scopes []string) []string {
	seen := make(map[string]bool, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
