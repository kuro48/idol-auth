package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

var (
	ErrInvalidAppName                 = errors.New("invalid app name")
	ErrInvalidAppSlug                 = errors.New("invalid app slug")
	ErrInvalidAppType                 = errors.New("invalid app type")
	ErrInvalidPartyType               = errors.New("invalid party type")
	ErrAppNotFound                    = errors.New("app not found")
	ErrAppDisabled                    = errors.New("app is disabled")
	ErrInvalidClientName              = errors.New("invalid client name")
	ErrInvalidClientType              = errors.New("invalid client type")
	ErrInvalidTokenEndpointAuthMethod = errors.New("invalid token endpoint auth method")
	ErrInvalidRedirectURI             = errors.New("invalid redirect uri")
	ErrRedirectURIsRequired           = errors.New("redirect uris are required")
	ErrRedirectURIsNotAllowed         = errors.New("redirect uris are not allowed for this app type")
	ErrOpenIDScopeRequired            = errors.New("openid scope is required")
	ErrConfidentialClientRequired     = errors.New("confidential client is required")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type AppRepository interface {
	Create(ctx context.Context, app App) (App, error)
	List(ctx context.Context) ([]App, error)
	GetByID(ctx context.Context, id uuid.UUID) (App, error)
}

type OIDCClientRepository interface {
	Create(ctx context.Context, client OIDCClient) (OIDCClient, error)
	ListByAppID(ctx context.Context, appID uuid.UUID) ([]OIDCClient, error)
}

type ClientProvisioner interface {
	CreateClient(ctx context.Context, spec ClientProvisionSpec) (ProvisionedClient, error)
	DeleteClient(ctx context.Context, clientID string) error
}

type Service struct {
	apps        AppRepository
	clients     OIDCClientRepository
	auditLogs   audit.Repository
	provisioner ClientProvisioner
	now         func() time.Time
}

type CreateAppInput struct {
	Name        string
	Slug        string
	Type        AppType
	PartyType   PartyType
	Description string
	ActorID     string
}

type CreateOIDCClientInput struct {
	Name                    string
	ClientType              ClientType
	TokenEndpointAuthMethod string
	RedirectURIs            []string
	PostLogoutRedirectURIs  []string
	Scopes                  []string
	ActorID                 string
}

type ClientRegistration struct {
	Client       OIDCClient
	ClientSecret string
}

type ClientProvisionSpec struct {
	HydraClientID           string
	Name                    string
	GrantTypes              []string
	ResponseTypes           []string
	RedirectURIs            []string
	PostLogoutRedirectURIs  []string
	Scopes                  []string
	TokenEndpointAuthMethod string
	PKCERequired            bool
	SkipConsent             bool
}

type ProvisionedClient struct {
	HydraClientID   string
	ClientSecret    string
	ClientSecretSet bool
}

type auditMetadata struct {
	AppSlug    string `json:"app_slug,omitempty"`
	ClientName string `json:"client_name,omitempty"`
}

func NewService(apps AppRepository, clients OIDCClientRepository, auditLogs audit.Repository, provisioner ClientProvisioner, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		apps:        apps,
		clients:     clients,
		auditLogs:   auditLogs,
		provisioner: provisioner,
		now:         now,
	}
}

func (s *Service) CreateApp(ctx context.Context, input CreateAppInput) (App, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 {
		return App{}, ErrInvalidAppName
	}

	slug := strings.TrimSpace(strings.ToLower(input.Slug))
	if slug == "" {
		slug = slugifyName(name)
	}
	if !slugPattern.MatchString(slug) {
		return App{}, ErrInvalidAppSlug
	}

	appType := normalizeAppType(input.Type)
	if !isValidAppType(appType) {
		return App{}, ErrInvalidAppType
	}
	if !isValidPartyType(input.PartyType) {
		return App{}, ErrInvalidPartyType
	}

	now := s.now().UTC()
	created, err := s.apps.Create(ctx, App{
		ID:          uuid.New(),
		Name:        name,
		Slug:        slug,
		Type:        appType,
		PartyType:   input.PartyType,
		Status:      AppStatusActive,
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   input.ActorID,
		UpdatedBy:   input.ActorID,
	})
	if err != nil {
		return App{}, err
	}

	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "app.created",
		ActorType:  audit.ActorTypeAdminClient,
		ActorID:    input.ActorID,
		TargetType: audit.TargetTypeApp,
		TargetID:   created.ID.String(),
		Result:     audit.ResultSuccess,
		Metadata:   marshalAuditMetadata(auditMetadata{AppSlug: created.Slug}),
		OccurredAt: now,
	})

	return created, nil
}

func (s *Service) ListApps(ctx context.Context) ([]App, error) {
	return s.apps.List(ctx)
}

func (s *Service) CreateOIDCClient(ctx context.Context, appID uuid.UUID, input CreateOIDCClientInput) (ClientRegistration, error) {
	parentApp, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return ClientRegistration{}, err
	}
	if parentApp.Status != AppStatusActive {
		return ClientRegistration{}, ErrAppDisabled
	}

	spec, client, err := s.buildClient(parentApp, input)
	if err != nil {
		return ClientRegistration{}, err
	}

	provisioned, err := s.provisioner.CreateClient(ctx, spec)
	if err != nil {
		return ClientRegistration{}, err
	}

	client.HydraClientID = provisioned.HydraClientID
	created, err := s.clients.Create(ctx, client)
	if err != nil {
		_ = s.provisioner.DeleteClient(ctx, provisioned.HydraClientID)
		return ClientRegistration{}, err
	}

	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "oidc_client.created",
		ActorType:  audit.ActorTypeAdminClient,
		ActorID:    input.ActorID,
		TargetType: audit.TargetTypeClient,
		TargetID:   created.ID.String(),
		Result:     audit.ResultSuccess,
		Metadata: marshalAuditMetadata(auditMetadata{
			AppSlug:    parentApp.Slug,
			ClientName: spec.Name,
		}),
		OccurredAt: s.now().UTC(),
	})

	return ClientRegistration{
		Client:       created,
		ClientSecret: provisioned.ClientSecret,
	}, nil
}

func (s *Service) ListOIDCClients(ctx context.Context, appID uuid.UUID) ([]OIDCClient, error) {
	if _, err := s.apps.GetByID(ctx, appID); err != nil {
		return nil, err
	}
	return s.clients.ListByAppID(ctx, appID)
}

func (s *Service) buildClient(parentApp App, input CreateOIDCClientInput) (ClientProvisionSpec, OIDCClient, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = parentApp.Name
	}
	if len(name) > 100 {
		return ClientProvisionSpec{}, OIDCClient{}, ErrInvalidClientName
	}

	clientType := input.ClientType
	if clientType == "" {
		clientType = defaultClientType(parentApp.Type)
	}
	if !isValidClientType(clientType) {
		return ClientProvisionSpec{}, OIDCClient{}, ErrInvalidClientType
	}

	redirectURIs, err := normalizeURIs(input.RedirectURIs, parentApp.Type)
	if err != nil {
		return ClientProvisionSpec{}, OIDCClient{}, err
	}
	postLogoutRedirectURIs, err := normalizeURIs(input.PostLogoutRedirectURIs, parentApp.Type)
	if err != nil {
		return ClientProvisionSpec{}, OIDCClient{}, err
	}
	scopes := normalizeScopes(input.Scopes)
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	authMethod, pkceRequired, grantTypes, responseTypes, err := s.resolveClientPolicy(parentApp.Type, clientType, input.TokenEndpointAuthMethod, redirectURIs, scopes)
	if err != nil {
		return ClientProvisionSpec{}, OIDCClient{}, err
	}

	now := s.now().UTC()
	hydraClientID := generateHydraClientID(parentApp.Slug)
	client := OIDCClient{
		ID:                      uuid.New(),
		HydraClientID:           hydraClientID,
		AppID:                   parentApp.ID,
		ClientType:              clientType,
		Status:                  ClientStatusActive,
		TokenEndpointAuthMethod: authMethod,
		PKCERequired:            pkceRequired,
		RedirectURIs:            redirectURIs,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs,
		Scopes:                  scopes,
		CreatedAt:               now,
		UpdatedAt:               now,
		CreatedBy:               input.ActorID,
		UpdatedBy:               input.ActorID,
	}

	spec := ClientProvisionSpec{
		HydraClientID:           hydraClientID,
		Name:                    name,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		RedirectURIs:            redirectURIs,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs,
		Scopes:                  scopes,
		TokenEndpointAuthMethod: authMethod,
		PKCERequired:            pkceRequired,
		SkipConsent:             parentApp.PartyType == PartyTypeFirst,
	}

	return spec, client, nil
}

func (s *Service) resolveClientPolicy(appType AppType, clientType ClientType, authMethod string, redirectURIs []string, scopes []string) (string, bool, []string, []string, error) {
	switch appType {
	case AppTypeWeb, AppTypeSPA, AppTypeNative:
		if len(redirectURIs) == 0 {
			return "", false, nil, nil, ErrRedirectURIsRequired
		}
		if !slices.Contains(scopes, "openid") {
			return "", false, nil, nil, ErrOpenIDScopeRequired
		}

		switch clientType {
		case ClientTypePublic:
			if authMethod == "" {
				authMethod = "none"
			}
			if authMethod != "none" {
				return "", false, nil, nil, ErrInvalidTokenEndpointAuthMethod
			}
			return authMethod, true, []string{"authorization_code", "refresh_token"}, []string{"code"}, nil
		case ClientTypeConfidential:
			if authMethod == "" {
				authMethod = "client_secret_basic"
			}
			if authMethod != "client_secret_basic" && authMethod != "client_secret_post" {
				return "", false, nil, nil, ErrInvalidTokenEndpointAuthMethod
			}
			return authMethod, true, []string{"authorization_code", "refresh_token"}, []string{"code"}, nil
		default:
			return "", false, nil, nil, ErrInvalidClientType
		}
	case AppTypeM2M:
		if len(redirectURIs) > 0 {
			return "", false, nil, nil, ErrRedirectURIsNotAllowed
		}
		if clientType != ClientTypeConfidential {
			return "", false, nil, nil, ErrConfidentialClientRequired
		}
		if authMethod == "" {
			authMethod = "client_secret_basic"
		}
		if authMethod != "client_secret_basic" && authMethod != "client_secret_post" {
			return "", false, nil, nil, ErrInvalidTokenEndpointAuthMethod
		}
		return authMethod, false, []string{"client_credentials"}, []string{"token"}, nil
	default:
		return "", false, nil, nil, ErrInvalidAppType
	}
}

func (s *Service) writeAudit(ctx context.Context, entry audit.Log) {
	if s.auditLogs == nil {
		return
	}
	_ = s.auditLogs.Write(ctx, entry)
}

func normalizeURIs(values []string, appType AppType) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed == nil {
			return nil, ErrInvalidRedirectURI
		}
		if parsed.Scheme == "" || parsed.Fragment != "" {
			return nil, ErrInvalidRedirectURI
		}

		switch parsed.Scheme {
		case "https":
			if parsed.Host == "" {
				return nil, ErrInvalidRedirectURI
			}
		case "http":
			if parsed.Host == "" || !isLoopbackHost(parsed.Hostname()) {
				return nil, ErrInvalidRedirectURI
			}
		default:
			if appType != AppTypeNative {
				return nil, ErrInvalidRedirectURI
			}
		}

		normalized := parsed.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	slices.Sort(out)
	return out, nil
}

func normalizeScopes(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		scope := strings.TrimSpace(value)
		if scope == "" || strings.Contains(scope, " ") {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	slices.Sort(out)
	return out
}

func isValidAppType(value AppType) bool {
	switch value {
	case AppTypeWeb, AppTypeSPA, AppTypeNative, AppTypeM2M:
		return true
	default:
		return false
	}
}

func isValidPartyType(value PartyType) bool {
	switch value {
	case PartyTypeFirst, PartyTypeThird:
		return true
	default:
		return false
	}
}

func isValidClientType(value ClientType) bool {
	switch value {
	case ClientTypePublic, ClientTypeConfidential:
		return true
	default:
		return false
	}
}

// normalizeAppType maps common aliases to canonical AppType values so callers
// don't need to memorise exact enum strings.
func normalizeAppType(value AppType) AppType {
	switch value {
	case "webapp", "server":
		return AppTypeWeb
	case "single-page", "single_page":
		return AppTypeSPA
	case "mobile":
		return AppTypeNative
	default:
		return value
	}
}

// defaultClientType infers the most appropriate ClientType for a given AppType
// when the caller does not specify one explicitly.
// SPA and native apps default to public (PKCE-only, no client secret).
// Web server and M2M apps default to confidential.
func defaultClientType(appType AppType) ClientType {
	switch appType {
	case AppTypeSPA, AppTypeNative:
		return ClientTypePublic
	default:
		return ClientTypeConfidential
	}
}

func marshalAuditMetadata(v auditMetadata) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func generateHydraClientID(slug string) string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d", slug, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(buf[:]))
}
