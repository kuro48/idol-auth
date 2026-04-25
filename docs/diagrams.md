# Diagrams

## ER図

```mermaid
erDiagram
    apps {
        UUID id PK
        TEXT name
        TEXT slug UK
        TEXT type "web|spa|native|m2m"
        TEXT party_type "first_party|third_party"
        TEXT status "active|disabled"
        TEXT description
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
        TEXT created_by
        TEXT updated_by
    }

    oidc_clients {
        UUID id PK
        TEXT hydra_client_id UK
        UUID app_id FK
        TEXT client_type "public|confidential"
        TEXT status "active|disabled|rotated"
        TEXT token_endpoint_auth_method
        BOOLEAN pkce_required
        TEXT redirect_uri_mode
        TEXT post_logout_redirect_uri_mode
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
        TEXT created_by
        TEXT updated_by
    }

    oidc_client_redirect_uris {
        UUID id PK
        UUID oidc_client_id FK
        TEXT uri
        TEXT kind "login_callback|post_logout_callback"
        TIMESTAMPTZ created_at
    }

    oidc_client_scopes {
        UUID id PK
        UUID oidc_client_id FK
        TEXT scope
        TIMESTAMPTZ created_at
    }

    admin_clients {
        UUID id PK
        TEXT hydra_client_id UK
        TEXT name
        TEXT status "active|disabled"
        TEXT description
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    audit_logs {
        UUID id PK
        TEXT event_type
        TEXT actor_type "admin_client|system"
        TEXT actor_id
        TEXT target_type "app|client|user"
        TEXT target_id
        TEXT result "success|failure"
        INET ip_address
        TEXT user_agent
        TEXT request_id
        JSONB metadata
        TIMESTAMPTZ occurred_at
    }

    operation_locks {
        TEXT key PK
        TEXT owner
        TIMESTAMPTZ expires_at
        TIMESTAMPTZ created_at
    }

    apps ||--o{ oidc_clients : "has"
    oidc_clients ||--o{ oidc_client_redirect_uris : "has"
    oidc_clients ||--o{ oidc_client_scopes : "has"
```

## クラス図

```mermaid
classDiagram
    %% ─── Domain Entities ───────────────────────────────────────
    class App {
        +UUID ID
        +string Name
        +string Slug
        +AppType Type
        +PartyType PartyType
        +AppStatus Status
        +string Description
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class OIDCClient {
        +UUID ID
        +string HydraClientID
        +UUID AppID
        +ClientType ClientType
        +ClientStatus Status
        +string TokenEndpointAuthMethod
        +bool PKCERequired
        +[]string RedirectURIs
        +[]string PostLogoutRedirectURIs
        +[]string Scopes
    }

    class AuditLog {
        +UUID ID
        +string EventType
        +ActorType ActorType
        +string ActorID
        +TargetType TargetType
        +string TargetID
        +Result Result
        +string IPAddress
        +string UserAgent
        +json.RawMessage Metadata
        +time.Time OccurredAt
    }

    class Identity {
        +string ID
        +string SchemaID
        +IdentityState State
        +string Email
        +string Phone
        +[]string Roles
    }

    %% ─── Repository Interfaces ──────────────────────────────────
    class AppRepository {
        <<interface>>
        +Create(ctx, App) App
        +List(ctx) []App
        +GetByID(ctx, UUID) App
    }

    class OIDCClientRepository {
        <<interface>>
        +Create(ctx, OIDCClient) OIDCClient
        +ListByAppID(ctx, UUID) []OIDCClient
    }

    class ClientProvisioner {
        <<interface>>
        +CreateClient(ctx, ClientProvisionSpec) ProvisionedClient
        +DeleteClient(ctx, clientID) error
    }

    class AuditRepository {
        <<interface>>
        +Write(ctx, Log) error
        +List(ctx, ListParams) []Log
    }

    class IdentityManager {
        <<interface>>
        +SetIdentityRoles(ctx, id, roles)
        +SearchIdentities(ctx, input) []Identity
        +DisableIdentity(ctx, input) Identity
        +EnableIdentity(ctx, input) Identity
        +RevokeIdentitySessions(ctx, id)
        +DeleteIdentity(ctx, id)
    }

    %% ─── Domain Services ────────────────────────────────────────
    class AppService {
        -AppRepository apps
        -OIDCClientRepository clients
        -AuditRepository auditLogs
        -ClientProvisioner provisioner
        +CreateApp(ctx, input) App
        +ListApps(ctx) []App
        +CreateOIDCClient(ctx, appID, input) ClientRegistration
        +ListOIDCClients(ctx, appID) []OIDCClient
    }

    class AdminService {
        -AppManager apps
        -IdentityManager identities
        -AuditRepository auditLogs
        +CreateApp(ctx, input) App
        +ListApps(ctx) []App
        +CreateOIDCClient(ctx, appID, input) ClientRegistration
        +SetIdentityRoles(ctx, input) []string
        +DisableIdentity(ctx, input) Identity
        +EnableIdentity(ctx, input) Identity
        +RevokeIdentitySessions(ctx, input)
        +DeleteIdentity(ctx, input)
        +ListAuditLogs(ctx, input) []AuditLog
    }

    %% ─── HTTP / Auth Layer ──────────────────────────────────────
    class AuthService {
        <<interface>>
        +HandleLogin(ctx, r, challenge) LoginFlowResult
        +HandleConsent(ctx, r, challenge) ConsentFlowResult
        +SubmitConsent(ctx, r, challenge, input) AuthFlowResult
        +HandleLogout(ctx, challenge) AuthFlowResult
        +CurrentSession(ctx, r) SessionView
    }

    class authService {
        -string baseURL
        -HydraAuthClient hydra
        -KratosAuthClient kratos
        +HandleLogin()
        +HandleConsent()
        +SubmitConsent()
        +HandleLogout()
        +CurrentSession()
    }

    class HydraAuthClient {
        <<interface>>
        +GetLoginRequest(ctx, challenge)
        +AcceptLoginRequest(ctx, challenge, subject)
        +GetConsentRequest(ctx, challenge)
        +AcceptConsentRequest(ctx, challenge, ...)
        +RejectConsentRequest(ctx, challenge, ...)
        +GetLogoutRequest(ctx, challenge)
        +AcceptLogoutRequest(ctx, challenge)
    }

    class KratosAuthClient {
        <<interface>>
        +ToSession(ctx, r) KratosSession
        +BrowserLoginURL(returnTo) string
        +BrowserSettingsURL(returnTo) string
    }

    %% ─── Relationships ──────────────────────────────────────────
    App "1" --> "many" OIDCClient : owns

    AppService ..|> AppRepository : uses
    AppService ..|> OIDCClientRepository : uses
    AppService ..|> ClientProvisioner : uses
    AppService ..|> AuditRepository : uses

    AdminService ..|> AuditRepository : uses
    AdminService --> AppService : delegates to
    AdminService ..|> IdentityManager : uses

    authService ..|> AuthService : implements
    authService ..|> HydraAuthClient : uses
    authService ..|> KratosAuthClient : uses

    AuditLog --> AuditRepository : persisted via
```
