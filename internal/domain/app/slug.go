package app

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphanumRun = regexp.MustCompile(`[^a-z0-9]+`)

// slugifyName converts a display name to a URL-safe slug deterministically.
// Non-alphanumeric characters (including Unicode) are replaced with hyphens;
// consecutive hyphens are collapsed; leading/trailing hyphens are stripped.
// Falls back to "app" when the result would be empty.
//
// Examples: "My SPA" → "my-spa", "  Hello  World!  " → "hello-world"
//
// Note: for idempotent provisioning (CI, IaC) always supply an explicit slug,
// because re-running with the same name produces the same derived slug — but
// the database unique constraint will reject duplicates without a helpful error.
func slugifyName(name string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '-'
	}, name)
	slug := nonAlphanumRun.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "app"
	}
	return slug
}
