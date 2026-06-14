package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/app"
)

func (s *server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Type        string `json:"type"`
		PartyType   string `json:"party_type"`
		Description string `json:"description"`
		// Top-level shorthand: provide redirect_uris here to create an OIDC
		// client inline without an explicit "client" block. client_type and
		// name are inferred automatically from the app. Takes effect only when
		// the "client" field is absent.
		RedirectURIs           []string `json:"redirect_uris"`
		PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
		Scopes                 []string `json:"scopes"`
		Client                 *struct {
			Name                    string   `json:"name"`
			ClientType              string   `json:"client_type"`
			TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
			RedirectURIs            []string `json:"redirect_uris"`
			PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
			Scopes                  []string `json:"scopes"`
		} `json:"client"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	actorID := adminActorIDFromContext(r.Context())
	created, err := s.adminSvc.CreateApp(r.Context(), app.CreateAppInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Type:        app.AppType(req.Type),
		PartyType:   app.PartyType(req.PartyType),
		Description: req.Description,
		ActorID:     actorID,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Determine whether to create an inline OIDC client.
	// Explicit "client" block takes precedence over top-level shorthand.
	var clientInput *app.CreateOIDCClientInput
	if req.Client != nil {
		clientInput = &app.CreateOIDCClientInput{
			Name:                    req.Client.Name,
			ClientType:              app.ClientType(req.Client.ClientType),
			TokenEndpointAuthMethod: req.Client.TokenEndpointAuthMethod,
			RedirectURIs:            req.Client.RedirectURIs,
			PostLogoutRedirectURIs:  req.Client.PostLogoutRedirectURIs,
			Scopes:                  req.Client.Scopes,
			ActorID:                 actorID,
		}
	} else if len(req.RedirectURIs) > 0 {
		clientInput = &app.CreateOIDCClientInput{
			RedirectURIs:           req.RedirectURIs,
			PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
			Scopes:                 req.Scopes,
			ActorID:                actorID,
		}
	}

	if clientInput == nil {
		w.Header().Set("Location", "/v1/admin/apps/"+created.ID.String())
		token, err := s.adminSvc.IssueManagementToken(r.Context(), created.ID, actorID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"app":              created,
			"management_token": token,
		})
		return
	}

	reg, err := s.adminSvc.CreateOIDCClient(r.Context(), created.ID, *clientInput)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	token, err := s.adminSvc.IssueManagementToken(r.Context(), created.ID, actorID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Location", "/v1/admin/apps/"+created.ID.String())
	writeJSON(w, http.StatusCreated, map[string]any{
		"app":              created,
		"client":           reg.Client,
		"client_secret":    reg.ClientSecret,
		"management_token": token,
	})
}

func (s *server) handleListApps(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	apps, err := s.adminSvc.ListApps(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list apps")
		return
	}
	if apps == nil {
		apps = []app.App{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": apps})
}

func (s *server) handleIssueManagementToken(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	token, err := s.adminSvc.IssueManagementToken(r.Context(), appID, adminActorIDFromContext(r.Context()))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":           appID.String(),
		"management_token": token,
	})
}

func (s *server) handleCreateOIDCClient(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}

	var req struct {
		Name                    string   `json:"name"`
		ClientType              string   `json:"client_type"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		RedirectURIs            []string `json:"redirect_uris"`
		PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
		Scopes                  []string `json:"scopes"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	created, err := s.adminSvc.CreateOIDCClient(r.Context(), appID, app.CreateOIDCClientInput{
		Name:                    req.Name,
		ClientType:              app.ClientType(req.ClientType),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		RedirectURIs:            req.RedirectURIs,
		PostLogoutRedirectURIs:  req.PostLogoutRedirectURIs,
		Scopes:                  req.Scopes,
		ActorID:                 adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Location", "/v1/admin/apps/"+appID.String()+"/clients/"+created.Client.ID.String())
	writeJSON(w, http.StatusCreated, map[string]any{
		"client":        created.Client,
		"client_secret": created.ClientSecret,
	})
}

func (s *server) handleSetAppPartyType(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	var body struct {
		PartyType app.PartyType `json:"party_type"`
	}
	if err := decodeJSON(w, r, &body); err != nil || body.PartyType == "" {
		writeError(w, http.StatusBadRequest, "party_type is required (first_party or third_party)")
		return
	}
	actorID, _ := r.Context().Value(adminActorIDKey).(string)
	updated, err := s.adminSvc.SetAppPartyType(r.Context(), appID, body.PartyType, actorID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *server) handleListOIDCClients(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}

	clients, err := s.adminSvc.ListOIDCClients(r.Context(), appID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": clients})
}

func (s *server) handleSearchIdentities(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	filter := admindomain.SearchIdentitiesInput{
		CredentialsIdentifier: strings.TrimSpace(r.URL.Query().Get("identifier")),
	}
	switch state := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("state"))); state {
	case "":
	case string(admindomain.IdentityStateActive):
		filter.State = admindomain.IdentityStateActive
	case string(admindomain.IdentityStateInactive):
		filter.State = admindomain.IdentityStateInactive
	default:
		writeError(w, http.StatusBadRequest, "invalid identity state")
		return
	}

	identities, err := s.adminSvc.SearchIdentities(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to search identities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": identities})
}

func (s *server) handleDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
		return
	}
	if err := s.adminSvc.DeleteIdentity(r.Context(), admindomain.DeleteIdentityInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	}); err != nil {
		writeError(w, http.StatusBadGateway, "failed to delete identity")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleRevokeIdentitySessions(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
		return
	}
	if err := s.adminSvc.RevokeIdentitySessions(r.Context(), admindomain.RevokeIdentitySessionsInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	}); err != nil {
		writeError(w, http.StatusBadGateway, "failed to revoke identity sessions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
		return
	}

	var req struct {
		State string    `json:"state"`
		Roles *[]string `json:"roles"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if req.State == "" && req.Roles == nil {
		writeError(w, http.StatusBadRequest, "at least one of 'state' or 'roles' is required")
		return
	}

	actorID := adminActorIDFromContext(r.Context())

	var identity admindomain.Identity
	var hasIdentity bool

	if req.State != "" {
		switch req.State {
		case string(admindomain.IdentityStateActive):
			identity, err = s.adminSvc.EnableIdentity(r.Context(), admindomain.EnableIdentityInput{
				IdentityID: identityID,
				ActorID:    actorID,
			})
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to enable identity")
				return
			}
			hasIdentity = true
		case string(admindomain.IdentityStateInactive):
			identity, err = s.adminSvc.DisableIdentity(r.Context(), admindomain.DisableIdentityInput{
				IdentityID: identityID,
				ActorID:    actorID,
			})
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to disable identity")
				return
			}
			hasIdentity = true
		default:
			writeError(w, http.StatusBadRequest, "state must be 'active' or 'inactive'")
			return
		}
	}

	if req.Roles != nil {
		roles, err := s.adminSvc.SetIdentityRoles(r.Context(), admindomain.SetIdentityRolesInput{
			IdentityID: identityID,
			Roles:      *req.Roles,
			ActorID:    actorID,
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to update identity roles")
			return
		}
		if hasIdentity {
			identity.Roles = roles
			writeJSON(w, http.StatusOK, identity)
		} else {
			writeJSON(w, http.StatusOK, map[string]any{
				"identity_id": identityID,
				"roles":       roles,
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, identity)
}

func (s *server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	limit := 0
	offset := 0
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}
	items, err := s.adminSvc.ListAuditLogs(r.Context(), admindomain.ListAuditLogsInput{
		ActorType:  strings.TrimSpace(r.URL.Query().Get("actor_type")),
		ActorID:    strings.TrimSpace(r.URL.Query().Get("actor_id")),
		TargetType: strings.TrimSpace(r.URL.Query().Get("target_type")),
		TargetID:   strings.TrimSpace(r.URL.Query().Get("target_id")),
		EventType:  strings.TrimSpace(r.URL.Query().Get("event_type")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list audit logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) resolveUserRef(ctx context.Context, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		return ref, nil
	}
	identities, err := s.adminSvc.SearchIdentities(ctx, admindomain.SearchIdentitiesInput{
		CredentialsIdentifier: ref,
	})
	if err != nil {
		return "", fmt.Errorf("search identity: %w", err)
	}
	switch len(identities) {
	case 0:
		return "", errUserNotFound
	case 1:
		return identities[0].ID, nil
	default:
		return "", errAmbiguousUserRef
	}
}

