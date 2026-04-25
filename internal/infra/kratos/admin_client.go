package kratos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
)

type AdminClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAdminClient(baseURL string) *AdminClient {
	return &AdminClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AdminClient) SetIdentityRoles(ctx context.Context, identityID string, roles []string) error {
	metadata, err := c.getMetadataPublic(ctx, identityID)
	if err != nil {
		return err
	}
	metadata["roles"] = roles

	payload, err := json.Marshal([]map[string]any{{
		"op":    "add",
		"path":  "/metadata_public",
		"value": metadata,
	}})
	if err != nil {
		return fmt.Errorf("marshal identity patch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/admin/identities/"+url.PathEscape(identityID), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build kratos patch identity request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call kratos patch identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "patch identity", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return fmt.Errorf("kratos patch identity returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *AdminClient) SearchIdentities(ctx context.Context, input admindomain.SearchIdentitiesInput) ([]admindomain.Identity, error) {
	query := url.Values{}
	if identifier := strings.TrimSpace(input.CredentialsIdentifier); identifier != "" {
		query.Set("credentials_identifier", identifier)
	}
	switch input.State {
	case admindomain.IdentityStateActive:
		query.Set("active", "true")
	case admindomain.IdentityStateInactive:
		query.Set("active", "false")
	}

	endpoint := c.baseURL + "/admin/identities"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build kratos list identities request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call kratos list identities: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("kratos list identities returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded []struct {
		ID             string `json:"id"`
		SchemaID       string `json:"schema_id"`
		State          string `json:"state"`
		MetadataPublic struct {
			Roles []string `json:"roles"`
		} `json:"metadata_public"`
		Traits struct {
			PrimaryIdentifierType string `json:"primary_identifier_type"`
			Email                 string `json:"email"`
			Phone                 string `json:"phone"`
		} `json:"traits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode kratos list identities response: %w", err)
	}

	identities := make([]admindomain.Identity, 0, len(decoded))
	for _, identity := range decoded {
		identities = append(identities, admindomain.Identity{
			ID:                    identity.ID,
			SchemaID:              identity.SchemaID,
			State:                 admindomain.IdentityState(identity.State),
			Email:                 identity.Traits.Email,
			Phone:                 identity.Traits.Phone,
			PrimaryIdentifierType: identity.Traits.PrimaryIdentifierType,
			Roles:                 append([]string(nil), identity.MetadataPublic.Roles...),
		})
	}
	return identities, nil
}

func (c *AdminClient) DisableIdentity(ctx context.Context, input admindomain.DisableIdentityInput) (admindomain.Identity, error) {
	return c.patchIdentityState(ctx, strings.TrimSpace(input.IdentityID), admindomain.IdentityStateInactive)
}

func (c *AdminClient) EnableIdentity(ctx context.Context, input admindomain.EnableIdentityInput) (admindomain.Identity, error) {
	return c.patchIdentityState(ctx, strings.TrimSpace(input.IdentityID), admindomain.IdentityStateActive)
}

func (c *AdminClient) RevokeIdentitySessions(ctx context.Context, identityID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/admin/identities/"+url.PathEscape(strings.TrimSpace(identityID))+"/sessions", nil)
	if err != nil {
		return fmt.Errorf("build kratos revoke identity sessions request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call kratos revoke identity sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "revoke identity sessions", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return fmt.Errorf("kratos revoke identity sessions returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *AdminClient) patchIdentityState(ctx context.Context, identityID string, state admindomain.IdentityState) (admindomain.Identity, error) {
	payload, err := json.Marshal([]map[string]any{{
		"op":    "replace",
		"path":  "/state",
		"value": string(state),
	}})
	if err != nil {
		return admindomain.Identity{}, fmt.Errorf("marshal identity state patch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/admin/identities/"+url.PathEscape(identityID), bytes.NewReader(payload))
	if err != nil {
		return admindomain.Identity{}, fmt.Errorf("build kratos patch identity state request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return admindomain.Identity{}, fmt.Errorf("call kratos patch identity state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "patch identity state", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return admindomain.Identity{}, fmt.Errorf("kratos patch identity state returned status %d", resp.StatusCode)
	}

	var decoded struct {
		ID       string `json:"id"`
		SchemaID string `json:"schema_id"`
		State    string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return admindomain.Identity{}, fmt.Errorf("decode kratos patch identity state response: %w", err)
	}

	return admindomain.Identity{
		ID:       decoded.ID,
		SchemaID: decoded.SchemaID,
		State:    admindomain.IdentityState(decoded.State),
	}, nil
}

func (c *AdminClient) DeleteIdentity(ctx context.Context, identityID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/admin/identities/"+url.PathEscape(strings.TrimSpace(identityID)), nil)
	if err != nil {
		return fmt.Errorf("build kratos delete identity request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call kratos delete identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "delete identity", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return fmt.Errorf("kratos delete identity returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *AdminClient) getMetadataPublic(ctx context.Context, identityID string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admin/identities/"+url.PathEscape(identityID), nil)
	if err != nil {
		return nil, fmt.Errorf("build kratos get identity request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call kratos get identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "get identity", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return nil, fmt.Errorf("kratos get identity returned status %d", resp.StatusCode)
	}

	var decoded struct {
		MetadataPublic map[string]any `json:"metadata_public"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode kratos get identity response: %w", err)
	}
	if decoded.MetadataPublic == nil {
		return map[string]any{}, nil
	}
	return decoded.MetadataPublic, nil
}
