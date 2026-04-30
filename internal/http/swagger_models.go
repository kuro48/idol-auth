package http

import "github.com/ryunosukekurokawa/idol-auth/internal/domain/app"

type swaggerStatusResponse struct {
	Status string `json:"status" example:"ok"`
}

type swaggerErrorResponse struct {
	Error string `json:"error" example:"invalid json body"`
}

type swaggerLogoutStartResponse struct {
	LogoutURL string `json:"logout_url" example:"http://localhost:4444/oauth2/sessions/logout"`
}

type swaggerThemePreferenceRequest struct {
	OshiColor string `json:"oshi_color" example:"#b2b2ff"`
}

type swaggerRolesUpdateRequest struct {
	Roles []string `json:"roles" example:"admin,support"`
}

type swaggerRolesUpdateResponse struct {
	IdentityID string   `json:"identity_id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	Roles      []string `json:"roles" example:"admin,support"`
}

type swaggerPatchUserRequest struct {
	State string    `json:"state,omitempty" example:"active"`
	Roles *[]string `json:"roles,omitempty"`
}

type swaggerCreateOIDCClientRequest struct {
	Name                    string   `json:"name" example:"Demo Browser Client"`
	ClientType              string   `json:"client_type,omitempty" example:"public"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" example:"none"`
	RedirectURIs            []string `json:"redirect_uris" example:"http://localhost:3002/callback"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty" example:"http://localhost:3002/"`
	Scopes                  []string `json:"scopes,omitempty" example:"openid,profile,email"`
}

type swaggerCreateOIDCClientResponse struct {
	Client       app.OIDCClient `json:"client"`
	ClientSecret string         `json:"client_secret" example:""`
}

type swaggerInlineClientCreateRequest struct {
	Name                    string   `json:"name,omitempty" example:"Demo Browser Client"`
	ClientType              string   `json:"client_type,omitempty" example:"confidential"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" example:"client_secret_basic"`
	RedirectURIs            []string `json:"redirect_uris" example:"https://example.com/callback"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty" example:"https://example.com/"`
	Scopes                  []string `json:"scopes,omitempty" example:"openid,email"`
}

type swaggerCreateAppRequest struct {
	Name                   string                            `json:"name" example:"Demo Web App"`
	Slug                   string                            `json:"slug,omitempty" example:"demo-web"`
	Type                   string                            `json:"type" example:"web"`
	PartyType              string                            `json:"party_type" example:"first_party"`
	Description            string                            `json:"description,omitempty" example:"Internal demo app"`
	RedirectURIs           []string                          `json:"redirect_uris,omitempty" example:"https://example.com/callback"`
	PostLogoutRedirectURIs []string                          `json:"post_logout_redirect_uris,omitempty" example:"https://example.com/"`
	Scopes                 []string                          `json:"scopes,omitempty" example:"openid,email"`
	Client                 *swaggerInlineClientCreateRequest `json:"client,omitempty"`
}

type swaggerCreateAppWithClientResponse struct {
	App             app.App        `json:"app"`
	Client          app.OIDCClient `json:"client"`
	ClientSecret    string         `json:"client_secret" example:""`
	ManagementToken string         `json:"management_token" example:"iat_0123456789abcdef"`
}

type swaggerCreateAppResponse struct {
	App             app.App `json:"app"`
	ManagementToken string  `json:"management_token" example:"iat_0123456789abcdef"`
}

type swaggerManagementTokenResponse struct {
	AppID           string `json:"app_id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	ManagementToken string `json:"management_token" example:"iat_0123456789abcdef"`
}

type swaggerAppListResponse struct {
	Items []app.App `json:"items"`
}

type swaggerOIDCClientListResponse struct {
	Items []app.OIDCClient `json:"items"`
}

type swaggerIdentity struct {
	ID                    string   `json:"id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	SchemaID              string   `json:"schema_id,omitempty" example:"default"`
	State                 string   `json:"state" example:"active"`
	Email                 string   `json:"email,omitempty" example:"user@example.com"`
	Phone                 string   `json:"phone,omitempty" example:"09012345678"`
	PrimaryIdentifierType string   `json:"primary_identifier_type,omitempty" example:"email"`
	Roles                 []string `json:"roles,omitempty" example:"viewer,admin"`
}

type swaggerIdentityListResponse struct {
	Items []swaggerIdentity `json:"items"`
}

type swaggerAuditLog struct {
	ID         string `json:"ID" example:"4bba3a6f-19ea-4b2d-9341-61eeb31b9cc2"`
	EventType  string `json:"EventType" example:"oidc_client.created"`
	ActorType  string `json:"ActorType" example:"admin_client"`
	ActorID    string `json:"ActorID" example:"bootstrap-admin"`
	TargetType string `json:"TargetType" example:"client"`
	TargetID   string `json:"TargetID" example:"f4e3f55f-bb7a-4787-a4fb-b346b6dc1a82"`
	Result     string `json:"Result" example:"success"`
	IPAddress  string `json:"IPAddress" example:"192.0.2.1"`
	UserAgent  string `json:"UserAgent" example:"curl/8.7.1"`
	RequestID  string `json:"RequestID" example:"req-123"`
	Metadata   string `json:"Metadata" example:"{}"`
	OccurredAt string `json:"OccurredAt" example:"2026-04-25T10:05:00Z"`
}

type swaggerAuditLogListResponse struct {
	Items []swaggerAuditLog `json:"items"`
}

type swaggerAccountOverviewResponse struct {
	Authenticated   bool                    `json:"authenticated"`
	Subject         string                  `json:"subject,omitempty" example:"user-123"`
	IdentityID      string                  `json:"identity_id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	Email           string                  `json:"email,omitempty" example:"user@example.com"`
	Memberships     []swaggerAppMembership  `json:"memberships"`
	DeletionRequest *swaggerDeletionRequest `json:"deletion_request,omitempty"`
}

type swaggerMembershipListResponse struct {
	App   app.App                `json:"app"`
	Items []swaggerAppMembership `json:"items"`
}

type swaggerDeletionRequestEnvelope struct {
	Request *swaggerDeletionRequest `json:"request,omitempty"`
}

type swaggerScheduleDeletionRequest struct {
	Reason string `json:"reason,omitempty" example:"user_requested"`
}

type swaggerAppMembership struct {
	ID         string `json:"id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	AppID      string `json:"app_id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	AppSlug    string `json:"app_slug" example:"idol-web"`
	AppName    string `json:"app_name" example:"Idol Web"`
	PartyType  string `json:"party_type" example:"first_party"`
	IdentityID string `json:"identity_id" example:"identity-123"`
	Status     string `json:"status" example:"active"`
	Profile    string `json:"profile,omitempty" example:"{}"`
	CreatedAt  string `json:"created_at" example:"2026-04-30T00:00:00Z"`
	UpdatedAt  string `json:"updated_at" example:"2026-04-30T00:00:00Z"`
	CreatedBy  string `json:"created_by,omitempty" example:"identity-123"`
	UpdatedBy  string `json:"updated_by,omitempty" example:"identity-123"`
}

type swaggerDeletionRequest struct {
	ID           string  `json:"id" example:"8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73"`
	IdentityID   string  `json:"identity_id" example:"identity-123"`
	Status       string  `json:"status" example:"scheduled"`
	Reason       string  `json:"reason,omitempty" example:"user_requested"`
	RequestedAt  string  `json:"requested_at" example:"2026-04-30T00:00:00Z"`
	ScheduledFor string  `json:"scheduled_for" example:"2026-05-07T00:00:00Z"`
	CancelledAt  *string `json:"cancelled_at,omitempty" example:"2026-05-01T00:00:00Z"`
	CompletedAt  *string `json:"completed_at,omitempty" example:"2026-05-07T00:00:00Z"`
	LastActorID  string  `json:"last_actor_id,omitempty" example:"identity-123"`
}
