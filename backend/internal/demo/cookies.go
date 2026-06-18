package demo

import (
	"strings"
)

// FilterOryCookies returns only Ory/Kratos cookie pairs. Portal endpoints call
// Kratos with the browser's cookies, but unrelated app cookies must not be
// forwarded to the upstream identity service.
func FilterOryCookies(cookieHeader string) string {
	parts := strings.Split(cookieHeader, ";")
	ory := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, "ory_") || strings.HasPrefix(trimmed, "csrf_token_") {
			ory = append(ory, trimmed)
		}
	}
	return strings.Join(ory, "; ")
}
