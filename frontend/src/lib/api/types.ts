// App types (snake_case JSON tags from app.go)
export interface App {
  id: string
  name: string
  slug: string
  type: string
  party_type: string
  status: string
  description: string
  created_at: string
  updated_at: string
  created_by: string
}

export interface OIDCClient {
  id: string
  hydra_client_id: string
  app_id: string
  client_type: string
  status: string
  token_endpoint_auth_method: string
  pkce_required: boolean
  redirect_uris: string[]
  post_logout_redirect_uris: string[]
  scopes: string[]
  created_at: string
  updated_at: string
}

// AppRequest (no JSON tags in appreg.go → PascalCase fields)
export interface AppRequest {
  ID: string
  IdentityID: string
  Status: string
  Name: string
  Slug: string
  Type: string
  Description: string
  HomepageURL: string
  PrivacyPolicyURL: string
  TermsURL: string
  ContactEmail: string
  Organization: string
  Purpose: string
  RedirectURIs: string[]
  PostLogoutRedirectURIs: string[]
  Scopes: string[]
  ReviewerID: string
  ReviewerNote: string
  DecidedAt: string | null
  CreatedAppID: string | null
  CreatedClientID: string | null
  Version: number
  CreatedAt: string
  UpdatedAt: string
}

// Identity (snake_case JSON tags from identity.go)
export interface Identity {
  id: string
  schema_id?: string
  state: string
  email?: string
  phone?: string
  primary_identifier_type?: string
  roles?: string[]
}

// AuditLog (no JSON tags in audit.go → PascalCase)
export interface AuditLog {
  ID: string
  EventType: string
  ActorType: string
  ActorID: string
  TargetType: string
  TargetID: string
  Result: string
  IPAddress: string
  UserAgent: string
  RequestID: string
  Metadata: unknown
  OccurredAt: string
}

// Account types (snake_case from account.go)
export interface AppMembership {
  id: string
  app_id: string
  app_slug: string
  app_name: string
  party_type: string
  identity_id: string
  status: string
  created_at: string
  updated_at: string
}

export interface DeletionRequest {
  id: string
  identity_id: string
  status: string
  reason?: string
  requested_at: string
  scheduled_for: string
  cancelled_at?: string
  completed_at?: string
}

// Session (from router.go SessionView)
export interface SessionView {
  authenticated: boolean
  subject?: string
  identity_id?: string
  email?: string
  display_name?: string
  roles?: string[]
  oshi_color?: string
  authenticator_assurance_level?: string
}

// Paginated list wrappers
export interface ListResponse<T> {
  items: T[]
}
