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

// AppRequest (snake_case JSON tags from appreg.go)
export interface AppRequest {
  id: string
  identity_id: string
  status: string
  name: string
  slug: string
  type: string
  description: string
  homepage_url: string
  privacy_policy_url: string
  terms_url: string
  contact_email: string
  organization: string
  purpose: string
  redirect_uris: string[]
  post_logout_redirect_uris: string[]
  scopes: string[]
  reviewer_id: string
  reviewer_note: string
  decided_at: string | null
  created_app_id: string | null
  created_client_id: string | null
  version: number
  created_at: string
  updated_at: string
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

// AuditLog (snake_case JSON tags from audit.go)
export interface AuditLog {
  id: string
  event_type: string
  actor_type: string
  actor_id: string
  target_type: string
  target_id: string
  result: string
  ip_address: string
  user_agent: string
  request_id: string
  metadata: unknown
  occurred_at: string
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
