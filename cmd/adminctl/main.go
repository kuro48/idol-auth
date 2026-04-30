package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

const requestTimeout = 10 * time.Second

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		slog.Error("adminctl failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(stderr, nil)))

	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("command is required")
	}

	switch args[0] {
	case "set-roles":
		return runSetRoles(args[1:], stdout)
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runSetRoles(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("set-roles", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	baseURL := fs.String("base-url", envOrDefault("ADMIN_API_BASE_URL", "http://localhost:8080"), "admin API base URL")
	token := fs.String("token", strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_TOKEN")), "admin bootstrap token")
	identityID := fs.String("identity-id", "", "Kratos identity ID")
	rolesFlag := fs.String("roles", "", "comma-separated roles")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*identityID) == "" {
		return errors.New("identity-id is required")
	}
	if strings.TrimSpace(*token) == "" {
		return errors.New("token is required")
	}

	roles := normalizeRoles(strings.Split(*rolesFlag, ","))
	if roles == nil {
		roles = []string{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	result, err := setRoles(ctx, *baseURL, *token, strings.TrimSpace(*identityID), roles)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}

func setRoles(ctx context.Context, baseURL, token, identityID string, roles []string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]any{"roles": roles})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, strings.TrimRight(baseURL, "/")+"/v1/admin/users/"+identityID, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("admin api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return decoded, nil
}

func normalizeRoles(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		role := strings.TrimSpace(strings.ToLower(value))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	slices.Sort(out)
	return out
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  adminctl set-roles --identity-id <kratos-identity-id> --roles admin,platform-operator [--base-url http://localhost:8080] [--token <bootstrap-token>]")
}
