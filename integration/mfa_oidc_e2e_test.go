package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalAuthMFAOIDCE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run local E2E tests")
	}

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	script := filepath.Join(root, "scripts", "smoke-local-auth.sh")
	cmd := exec.Command(script)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"APP_URL="+envOrDefault("APP_URL", "http://localhost:3002"),
		"AUTH_URL="+envOrDefault("AUTH_URL", "http://localhost:8080"),
		"KRATOS_BROWSER_URL="+envOrDefault("KRATOS_BROWSER_URL", "http://localhost:4433"),
		"MAILPIT_URL="+envOrDefault("MAILPIT_URL", "http://localhost:8025"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("smoke-local-auth.sh failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Smoke test passed") {
		t.Fatalf("unexpected smoke output:\n%s", output)
	}
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(dir), nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
