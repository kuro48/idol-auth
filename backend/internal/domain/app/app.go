// Package app defines the core entities for registered applications and their OIDC clients.
package app

import (
	"time"

	"github.com/google/uuid"
)

type AppType string
type PartyType string
type AppStatus string
type ClientType string
type ClientStatus string

const (
	AppTypeWeb    AppType = "web"
	AppTypeSPA    AppType = "spa"
	AppTypeNative AppType = "native"
	AppTypeM2M    AppType = "m2m"

	PartyTypeFirst PartyType = "first_party"
	PartyTypeThird PartyType = "third_party"

	AppStatusActive   AppStatus = "active"
	AppStatusDisabled AppStatus = "disabled"

	ClientTypePublic       ClientType = "public"
	ClientTypeConfidential ClientType = "confidential"

	ClientStatusActive   ClientStatus = "active"
	ClientStatusDisabled ClientStatus = "disabled"
	ClientStatusRotated  ClientStatus = "rotated"
)

// App represents an application registered in the auth platform.
type App struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Type        AppType   `json:"type"`
	PartyType   PartyType `json:"party_type"`
	Status      AppStatus `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
}

// OIDCClient represents the metadata of an OAuth2/OIDC client linked to an App.
type OIDCClient struct {
	ID                      uuid.UUID    `json:"id"`
	HydraClientID           string       `json:"hydra_client_id"`
	AppID                   uuid.UUID    `json:"app_id"`
	ClientType              ClientType   `json:"client_type"`
	Status                  ClientStatus `json:"status"`
	TokenEndpointAuthMethod string       `json:"token_endpoint_auth_method"`
	PKCERequired            bool         `json:"pkce_required"`
	RedirectURIs            []string     `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string     `json:"post_logout_redirect_uris"`
	Scopes                  []string     `json:"scopes"`
	CreatedAt               time.Time    `json:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at"`
	CreatedBy               string       `json:"created_by"`
	UpdatedBy               string       `json:"updated_by"`
}
