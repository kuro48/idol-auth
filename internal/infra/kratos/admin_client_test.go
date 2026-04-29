package kratos

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
)

func TestAdminClientSearchIdentitiesBuildsFiltersAndParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/identities" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("credentials_identifier"); got != "user@example.com" {
			t.Fatalf("expected credentials_identifier query, got %q", got)
		}
		if got := r.URL.Query().Get("active"); got != "true" {
			t.Fatalf("expected active=true query, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":        "identity-123",
			"schema_id": "default",
			"state":     "active",
			"traits": map[string]any{
				"email":                   "user@example.com",
				"primary_identifier_type": "email",
			},
			"metadata_public": map[string]any{
				"roles": []string{"admin"},
			},
		}})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	identities, err := client.SearchIdentities(context.Background(), admindomain.SearchIdentitiesInput{
		CredentialsIdentifier: "user@example.com",
		State:                 admindomain.IdentityStateActive,
	})
	if err != nil {
		t.Fatalf("SearchIdentities() error = %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
	if identities[0].Email != "user@example.com" || identities[0].State != admindomain.IdentityStateActive {
		t.Fatalf("unexpected identities: %+v", identities)
	}
}

func TestAdminClientDisableIdentityPatchesState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/admin/identities/identity-123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var patch []map[string]any
		if err := json.Unmarshal(body, &patch); err != nil {
			t.Fatalf("unmarshal patch: %v", err)
		}
		if len(patch) != 1 || patch[0]["path"] != "/state" || patch[0]["value"] != "inactive" {
			t.Fatalf("unexpected patch payload: %v", patch)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        "identity-123",
			"schema_id": "default",
			"state":     "inactive",
		})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	identity, err := client.DisableIdentity(context.Background(), admindomain.DisableIdentityInput{IdentityID: "identity-123"})
	if err != nil {
		t.Fatalf("DisableIdentity() error = %v", err)
	}
	if identity.State != admindomain.IdentityStateInactive {
		t.Fatalf("expected inactive state, got %+v", identity)
	}
}

func TestAdminClientEnableIdentityPatchesState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/admin/identities/identity-123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var patch []map[string]any
		if err := json.Unmarshal(body, &patch); err != nil {
			t.Fatalf("unmarshal patch: %v", err)
		}
		if len(patch) != 1 || patch[0]["path"] != "/state" || patch[0]["value"] != "active" {
			t.Fatalf("unexpected patch payload: %v", patch)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        "identity-123",
			"schema_id": "default",
			"state":     "active",
		})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	identity, err := client.EnableIdentity(context.Background(), admindomain.EnableIdentityInput{IdentityID: "identity-123"})
	if err != nil {
		t.Fatalf("EnableIdentity() error = %v", err)
	}
	if identity.State != admindomain.IdentityStateActive {
		t.Fatalf("expected active state, got %+v", identity)
	}
}

func TestAdminClientDeleteIdentityCallsDeleteEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/admin/identities/identity-123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	if err := client.DeleteIdentity(context.Background(), "identity-123"); err != nil {
		t.Fatalf("DeleteIdentity() error = %v", err)
	}
}

func TestAdminClientRevokeIdentitySessionsCallsDeleteEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/admin/identities/identity-123/sessions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	if err := client.RevokeIdentitySessions(context.Background(), "identity-123"); err != nil {
		t.Fatalf("RevokeIdentitySessions() error = %v", err)
	}
}

func TestAdminClientSetIdentityOshiColorPatchesMetadataPublic(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/admin/identities/identity-123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		switch requestCount {
		case 1:
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata_public": map[string]any{
					"roles": []string{"admin"},
				},
			})
		case 2:
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var patch []map[string]any
			if err := json.Unmarshal(body, &patch); err != nil {
				t.Fatalf("unmarshal patch: %v", err)
			}
			value, ok := patch[0]["value"].(map[string]any)
			if !ok {
				t.Fatalf("expected metadata_public patch value, got %T", patch[0]["value"])
			}
			if value["oshi_color"] != "#ffb2d8" {
				t.Fatalf("expected oshi_color to be set, got %#v", value["oshi_color"])
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request count %d", requestCount)
		}
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	if err := client.SetIdentityOshiColor(context.Background(), "identity-123", "#ffb2d8"); err != nil {
		t.Fatalf("SetIdentityOshiColor() error = %v", err)
	}
}
