package kratos

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuro48/idol-auth/internal/domain/account"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/profile"
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

func TestGetIdentityProfile_ReadsAllFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/admin/identities/identity-1" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "identity-1",
			"traits": map[string]any{
				"email":        "user@example.com",
				"phone":        "+81900000000",
				"display_name": "推し活太郎",
			},
			"metadata_public": map[string]any{
				"oshi_color": "#ffb2d8",
				"oshis": []map[string]any{
					{"idol_id": "member-01", "fan_since": "2019-04"},
					{"idol_id": "member-03"},
				},
				"avatar_url":       "https://example.com/avatar.png",
				"locale":           "ja-JP",
				"timezone":         "Asia/Tokyo",
				"badges":           []map[string]any{{"id": "top", "label": "Top"}},
				"primary_badge_id": "top",
			},
			"metadata_admin": map[string]any{
				"birthdate": "2000-01-02",
				"notification_preferences": map[string]any{
					"email_enabled":   true,
					"security_alerts": true,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	p, err := client.GetIdentityProfile(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("GetIdentityProfile() error = %v", err)
	}
	if p.IdentityID != "identity-1" {
		t.Errorf("IdentityID = %q, want identity-1", p.IdentityID)
	}
	if p.Email != "user@example.com" {
		t.Errorf("Email = %q, want user@example.com", p.Email)
	}
	if p.Phone != "+81900000000" {
		t.Errorf("Phone = %q, want +81900000000", p.Phone)
	}
	if p.DisplayName != "推し活太郎" {
		t.Errorf("DisplayName = %q, want 推し活太郎", p.DisplayName)
	}
	if p.OshiColor != "#ffb2d8" {
		t.Errorf("OshiColor = %q, want #ffb2d8", p.OshiColor)
	}
	if len(p.Oshis) != 2 || p.Oshis[0].IdolID != "member-01" || p.Oshis[0].FanSince != "2019-04" || p.Oshis[1].IdolID != "member-03" {
		t.Errorf("Oshis = %v, want [{member-01 2019-04} {member-03 }]", p.Oshis)
	}
	if p.AvatarURL != "https://example.com/avatar.png" || p.Locale != "ja-JP" || p.Timezone != "Asia/Tokyo" {
		t.Errorf("common profile fields mismatch: %+v", p)
	}
	if len(p.Badges) != 1 || p.PrimaryBadgeID != "top" {
		t.Errorf("badge fields mismatch: %+v", p)
	}
	if p.Birthdate != "2000-01-02" || !p.NotificationPreferences.EmailEnabled || !p.NotificationPreferences.SecurityAlerts {
		t.Errorf("private profile fields mismatch: %+v", p)
	}
}

func TestGetIdentityProfile_HandlesEmptyMetadataPublic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "identity-1",
			"traits": map[string]any{
				"email": "user@example.com",
			},
		})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	p, err := client.GetIdentityProfile(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("GetIdentityProfile() error = %v", err)
	}
	if p.OshiColor != "" || len(p.Oshis) != 0 {
		t.Errorf("expected zero metadata fields, got %+v", p)
	}
}

func TestGetIdentityProfile_Returns404AsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	_, err := client.GetIdentityProfile(context.Background(), "identity-1")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestUpdateIdentityProfile_MergesExistingMetadataAndPatches(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/admin/identities/identity-1" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		switch requestCount {
		case 1:
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET for request 1, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "identity-1",
				"traits": map[string]any{
					"email": "user@example.com",
				},
				"metadata_public": map[string]any{
					"roles": []string{"user"},
				},
			})
		case 2:
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH for request 2, got %s", r.Method)
			}
			var patch []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Fatalf("decode patch: %v", err)
			}
			var metaValue map[string]any
			var foundDisplayName bool
			for _, op := range patch {
				switch op["path"] {
				case "/metadata_public":
					metaValue, _ = op["value"].(map[string]any)
				case "/traits/display_name":
					foundDisplayName = true
					if op["value"] != "推し活太郎" {
						t.Errorf("display_name = %v, want 推し活太郎", op["value"])
					}
				}
			}
			if metaValue == nil {
				t.Fatal("expected /metadata_public patch op")
			}
			roles, _ := metaValue["roles"].([]any)
			if len(roles) != 1 || roles[0] != "user" {
				t.Errorf("roles not preserved in metadata_public: %v", metaValue["roles"])
			}
			if metaValue["oshi_color"] != "#ffb2d8" {
				t.Errorf("oshi_color = %v, want #ffb2d8", metaValue["oshi_color"])
			}
			oshis, _ := metaValue["oshis"].([]any)
			if len(oshis) != 1 {
				t.Errorf("oshis length = %d, want 1", len(oshis))
			}
			if metaValue["avatar_url"] != "https://example.com/avatar.png" {
				t.Errorf("avatar_url = %v", metaValue["avatar_url"])
			}
			if metaValue["locale"] != "ja-JP" || metaValue["timezone"] != "Asia/Tokyo" {
				t.Errorf("locale/timezone = %v/%v", metaValue["locale"], metaValue["timezone"])
			}
			if !foundDisplayName {
				t.Error("expected /traits/display_name patch op")
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request count %d", requestCount)
		}
	}))
	defer srv.Close()

	displayName := "推し活太郎"
	oshiColor := "#ffb2d8"
	oshis := []profile.OshiEntry{{IdolID: "member-01", FanSince: "2019-04"}}
	avatarURL := "https://example.com/avatar.png"
	locale := "ja-JP"
	timezone := "Asia/Tokyo"
	client := NewAdminClient(srv.URL)
	err := client.UpdateIdentityProfile(context.Background(), "identity-1", profile.UpdateInput{
		DisplayName: &displayName,
		OshiColor:   &oshiColor,
		Oshis:       &oshis,
		AvatarURL:   &avatarURL,
		Locale:      &locale,
		Timezone:    &timezone,
	})
	if err != nil {
		t.Fatalf("UpdateIdentityProfile() error = %v", err)
	}
}

func TestUpdateIdentityProfile_PatchesMetadataAdminForPrivateFields(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata_admin": map[string]any{"existing": true},
			})
		case 2:
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			var patch []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Fatalf("decode patch: %v", err)
			}
			if len(patch) != 1 || patch[0]["path"] != "/metadata_admin" {
				t.Fatalf("expected metadata_admin patch, got %+v", patch)
			}
			value, _ := patch[0]["value"].(map[string]any)
			if value["birthdate"] != "2000-01-02" {
				t.Fatalf("expected birthdate, got %+v", value)
			}
			prefs, _ := value["notification_preferences"].(map[string]any)
			if prefs["security_alerts"] != true {
				t.Fatalf("expected notification preferences, got %+v", prefs)
			}
			if value["existing"] != true {
				t.Fatalf("expected existing metadata preserved, got %+v", value)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request count %d", requestCount)
		}
	}))
	defer srv.Close()

	birthdate := "2000-01-02"
	prefs := profile.NotificationPreferences{EmailEnabled: true, SecurityAlerts: true}
	client := NewAdminClient(srv.URL)
	if err := client.UpdateIdentityProfile(context.Background(), "identity-1", profile.UpdateInput{
		Birthdate:               &birthdate,
		NotificationPreferences: &prefs,
	}); err != nil {
		t.Fatalf("UpdateIdentityProfile() error = %v", err)
	}
}

func TestUpdateIdentityProfile_SkipsNilFields(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Only display_name is set → no metadata GET needed, single PATCH only
		if requestCount != 1 {
			t.Fatalf("unexpected request count %d; expected only 1 request", requestCount)
		}
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		var patch []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			t.Fatalf("decode patch: %v", err)
		}
		for _, op := range patch {
			if op["path"] == "/metadata_public" {
				t.Error("unexpected /metadata_public op when no metadata fields are set")
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	displayName := "新しい名前"
	client := NewAdminClient(srv.URL)
	err := client.UpdateIdentityProfile(context.Background(), "identity-1", profile.UpdateInput{
		DisplayName: &displayName,
	})
	if err != nil {
		t.Fatalf("UpdateIdentityProfile() error = %v", err)
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

func TestAdminClientCreateSharedAccount_ReturnsNewIdentityOn201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/admin/identities" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		traits, _ := body["traits"].(map[string]any)
		if traits["email"] != "user@example.com" {
			t.Fatalf("expected email user@example.com, got %v", traits["email"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-identity-id"})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	result, err := client.CreateSharedAccount(context.Background(), account.RegisterIdentityInput{
		Email:       "user@example.com",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("CreateSharedAccount() error = %v", err)
	}
	if result.IdentityID != "new-identity-id" {
		t.Errorf("expected identity id new-identity-id, got %q", result.IdentityID)
	}
	if !result.IsNew {
		t.Error("expected IsNew=true for 201 response")
	}
}

func TestAdminClientCreateSharedAccount_ReturnsConflictOn409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	_, err := client.CreateSharedAccount(context.Background(), account.RegisterIdentityInput{
		Email: "user@example.com",
	})
	if !errors.Is(err, account.ErrSharedAccountAlreadyExists) {
		t.Fatalf("expected ErrSharedAccountAlreadyExists, got %v", err)
	}
}

func TestAdminClientCreateRecoveryLink_ReturnsLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/admin/recovery/link" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["identity_id"] != "identity-123" {
			t.Fatalf("expected identity_id identity-123, got %v", body["identity_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"recovery_link": "https://auth.example.com/self-service/recovery?token=abc123",
		})
	}))
	defer srv.Close()

	client := NewAdminClient(srv.URL)
	link, err := client.CreateRecoveryLink(context.Background(), "identity-123")
	if err != nil {
		t.Fatalf("CreateRecoveryLink() error = %v", err)
	}
	if link != "https://auth.example.com/self-service/recovery?token=abc123" {
		t.Errorf("unexpected recovery link: %q", link)
	}
}
