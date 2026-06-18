package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKratosConfigRequiresVerifiedEmailBeforeLogin(t *testing.T) {
	for _, name := range []string{"kratos.yml", "kratos.production.yml.tmpl"} {
		t.Run(name, func(t *testing.T) {
			body := readKratosDeployFile(t, name)

			assertContains(t, body, "verification:\n      enabled: true")
			assertContains(t, body, "verification:\n      enabled: true\n      ui_url:")
			assertContains(t, body, "      use: code")
			assertContains(t, body, "registration:\n      enabled: true")
			assertContains(t, body, "          hooks:\n            - hook: show_verification_ui")
			assertContains(t, body, "login:\n      ui_url:")
			assertContains(t, body, "      after:\n        password:\n          hooks:\n            - hook: require_verified_address")

			if strings.Contains(body, "- hook: session") {
				t.Fatal("registration must not issue a session before email verification")
			}
		})
	}
}

func TestKratosIdentitySchemaMarksEmailAsVerifiable(t *testing.T) {
	body := readKratosDeployFile(t, "identity.schema.json")
	var schema map[string]any
	if err := json.Unmarshal([]byte(body), &schema); err != nil {
		t.Fatalf("decode identity schema: %v", err)
	}

	email := nestedMap(t, schema, "properties", "traits", "properties", "email", "ory.sh/kratos")
	verification := nestedMap(t, email, "verification")
	if got := verification["via"]; got != "email" {
		t.Fatalf("expected email verification via email, got %v", got)
	}
}

func TestKratosIdentitySchemaOnlyAllowsEmailAsRegistrationIdentifier(t *testing.T) {
	body := readKratosDeployFile(t, "identity.schema.json")
	var schema map[string]any
	if err := json.Unmarshal([]byte(body), &schema); err != nil {
		t.Fatalf("decode identity schema: %v", err)
	}

	traits := nestedMap(t, schema, "properties", "traits", "properties")
	primaryIdentifierType := nestedMap(t, traits, "primary_identifier_type")
	enum, ok := primaryIdentifierType["enum"].([]any)
	if !ok {
		t.Fatalf("expected primary_identifier_type enum, got %T", primaryIdentifierType["enum"])
	}
	if len(enum) != 1 || enum[0] != "email" {
		t.Fatalf("expected primary_identifier_type enum to only allow email, got %v", enum)
	}

	phone := nestedMap(t, traits, "phone", "ory.sh/kratos")
	if _, ok := phone["credentials"]; ok {
		t.Fatal("phone must not be usable as a registration or login credential")
	}
	if _, ok := phone["verification"]; ok {
		t.Fatal("phone verification is disabled until an SMS provider is configured")
	}
}

func readKratosDeployFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "deploy", "kratos", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func assertContains(t *testing.T, body, needle string) {
	t.Helper()
	if !strings.Contains(body, needle) {
		t.Fatalf("expected config to contain %q", needle)
	}
}

func nestedMap(t *testing.T, root map[string]any, path ...string) map[string]any {
	t.Helper()
	current := root
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("expected %q to be an object in path %v", key, path)
		}
		current = next
	}
	return current
}
