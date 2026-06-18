export interface IdolAuthClientOptions {
  /** Base URL of the idol-auth server, e.g. "https://api.example.com" */
  baseUrl: string;
  /** Default fetch implementation (defaults to global fetch) */
  fetch?: typeof fetch;
}

// ── High-level redirect flow ─────────────────────────────────────────────────

/** Minimal storage interface; defaults to window.sessionStorage. */
export interface TxStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export interface LoginWithRedirectOptions {
  clientId: string;
  redirectUri: string;
  /** Defaults to "openid email profile offline_access". */
  scope?: string;
  /** Where the PKCE verifier and state are kept until the callback. Defaults to sessionStorage. */
  storage?: TxStorage;
  /** Override navigation; defaults to assigning window.location. Receives the authorization URL. */
  navigate?: (url: string) => void;
}

export interface HandleRedirectCallbackOptions {
  /** Callback URL to parse; defaults to window.location.href. */
  url?: string;
  /** Must match the storage used in loginWithRedirect. Defaults to sessionStorage. */
  storage?: TxStorage;
}

// ── Browser-mode ─────────────────────────────────────────────────────────────

export interface BrowserLoginParams {
  clientId: string;
  redirectUri: string;
  responseType?: string;
  scope?: string;
  state?: string;
  nonce?: string;
  codeChallenge?: string;
  codeChallengeMethod?: string;
}

export interface BrowserLogoutParams {
  idTokenHint?: string;
  postLogoutRedirectUri?: string;
  state?: string;
}

// ── API-mode token operations ─────────────────────────────────────────────────

export interface TokenRequest {
  grant_type: string;
  code?: string;
  redirect_uri?: string;
  client_id?: string;
  client_secret?: string;
  code_verifier?: string;
  refresh_token?: string;
  scope?: string;
  [key: string]: string | undefined;
}

export interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
  id_token?: string;
  scope?: string;
}

export interface RevokeRequest {
  token: string;
  client_id?: string;
  client_secret?: string;
  [key: string]: string | undefined;
}

export interface IntrospectRequest {
  token: string;
  client_id?: string;
  client_secret?: string;
  [key: string]: string | undefined;
}

export interface IntrospectResponse {
  active: boolean;
  sub?: string;
  client_id?: string;
  scope?: string;
  exp?: number;
  iat?: number;
  [key: string]: unknown;
}

export interface SessionResponse {
  active: boolean;
  subject?: string;
  email?: string;
  display_name?: string;
  roles?: string[];
  oshi_color?: string;
}

// ── Headless auth ─────────────────────────────────────────────────────────────

export interface RegisterRequest {
  email: string;
  password: string;
  display_name?: string;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface AuthResult {
  session_token?: string;
  identity_id: string;
  email: string;
}
