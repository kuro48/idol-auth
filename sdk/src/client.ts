import type {
  IdolAuthClientOptions,
  BrowserLoginParams,
  BrowserLogoutParams,
  TokenRequest,
  TokenResponse,
  RevokeRequest,
  IntrospectRequest,
  IntrospectResponse,
  SessionResponse,
  RegisterRequest,
  LoginRequest,
  AuthResult,
  TxStorage,
  LoginWithRedirectOptions,
  HandleRedirectCallbackOptions,
} from "./types.js";
import { generatePKCE } from "./pkce.js";

const DEFAULT_SCOPE = "openid email profile offline_access";
const TX_STORAGE_KEY = "idol_auth_tx";

interface LoginTransaction {
  codeVerifier: string;
  state: string;
  clientId: string;
  redirectUri: string;
}

function defaultStorage(): TxStorage {
  if (typeof window === "undefined" || !window.sessionStorage) {
    throw new Error(
      "idol-auth: sessionStorage is not available; pass a custom `storage` option"
    );
  }
  return window.sessionStorage;
}

export class IdolAuthClient {
  private readonly baseUrl: string;
  private readonly _fetch: typeof fetch;

  constructor(options: IdolAuthClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, "");
    this._fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  // ── Browser-mode helpers ──────────────────────────────────────────────────

  /**
   * Returns the URL to redirect the browser to for OAuth2 authorization.
   * Typically used as: `window.location.href = client.browserLoginUrl({...})`
   */
  browserLoginUrl(params: BrowserLoginParams): string {
    const q = new URLSearchParams({
      client_id: params.clientId,
      redirect_uri: params.redirectUri,
      response_type: params.responseType ?? "code",
      ...(params.scope && { scope: params.scope }),
      ...(params.state && { state: params.state }),
      ...(params.nonce && { nonce: params.nonce }),
      ...(params.codeChallenge && { code_challenge: params.codeChallenge }),
      ...(params.codeChallengeMethod && {
        code_challenge_method: params.codeChallengeMethod,
      }),
    });
    return `${this.baseUrl}/v1/public/browser/login?${q}`;
  }

  /**
   * Returns the URL to redirect the browser to for Kratos registration.
   */
  browserRegistrationUrl(returnTo?: string): string {
    const q = returnTo ? `?return_to=${encodeURIComponent(returnTo)}` : "";
    return `${this.baseUrl}/v1/public/browser/registration${q}`;
  }

  /**
   * Returns the URL to redirect the browser to for OIDC logout.
   */
  browserLogoutUrl(params?: BrowserLogoutParams): string {
    const entries: Record<string, string> = {};
    if (params?.idTokenHint) entries["id_token_hint"] = params.idTokenHint;
    if (params?.postLogoutRedirectUri)
      entries["post_logout_redirect_uri"] = params.postLogoutRedirectUri;
    if (params?.state) entries["state"] = params.state;
    const q = new URLSearchParams(entries).toString();
    return `${this.baseUrl}/v1/public/browser/logout${q ? "?" + q : ""}`;
  }

  // ── High-level redirect flow ──────────────────────────────────────────────

  /**
   * Starts the OAuth2 authorization code flow with PKCE: generates a
   * verifier/challenge pair and state, stores them, and navigates to the
   * authorization URL. Returns the URL (useful for testing or manual control).
   */
  async loginWithRedirect(options: LoginWithRedirectOptions): Promise<string> {
    const storage = options.storage ?? defaultStorage();
    const pkce = await generatePKCE();
    const state = crypto.randomUUID();

    const tx: LoginTransaction = {
      codeVerifier: pkce.codeVerifier,
      state,
      clientId: options.clientId,
      redirectUri: options.redirectUri,
    };
    storage.setItem(TX_STORAGE_KEY, JSON.stringify(tx));

    const url = this.browserLoginUrl({
      clientId: options.clientId,
      redirectUri: options.redirectUri,
      scope: options.scope ?? DEFAULT_SCOPE,
      state,
      codeChallenge: pkce.codeChallenge,
      codeChallengeMethod: pkce.codeChallengeMethod,
    });

    if (options.navigate) {
      options.navigate(url);
    } else if (typeof window !== "undefined") {
      window.location.assign(url);
    }
    return url;
  }

  /**
   * Completes the redirect flow on the callback page: validates state,
   * exchanges the authorization code for tokens using the stored PKCE
   * verifier, and clears the stored transaction.
   */
  async handleRedirectCallback(
    options?: HandleRedirectCallbackOptions
  ): Promise<TokenResponse> {
    const storage = options?.storage ?? defaultStorage();
    const href =
      options?.url ??
      (typeof window !== "undefined" ? window.location.href : undefined);
    if (!href) {
      throw new Error("idol-auth: no callback URL available; pass `url`");
    }

    const params = new URL(href).searchParams;
    const oauthError = params.get("error");
    if (oauthError) {
      storage.removeItem(TX_STORAGE_KEY);
      throw new IdolAuthCallbackError(
        oauthError,
        params.get("error_description") ?? ""
      );
    }
    const code = params.get("code");
    if (!code) {
      throw new Error("idol-auth: callback URL is missing the `code` parameter");
    }

    const raw = storage.getItem(TX_STORAGE_KEY);
    if (!raw) {
      throw new Error(
        "idol-auth: no login transaction found; did you call loginWithRedirect() in this browser session?"
      );
    }
    const tx = JSON.parse(raw) as LoginTransaction;
    if (params.get("state") !== tx.state) {
      storage.removeItem(TX_STORAGE_KEY);
      throw new Error("idol-auth: state mismatch; possible CSRF, login aborted");
    }
    storage.removeItem(TX_STORAGE_KEY);

    return this.token({
      grant_type: "authorization_code",
      code,
      redirect_uri: tx.redirectUri,
      client_id: tx.clientId,
      code_verifier: tx.codeVerifier,
    });
  }

  // ── Token operations ──────────────────────────────────────────────────────

  /** Exchange an authorization code (or refresh token) for tokens. */
  async token(req: TokenRequest): Promise<TokenResponse> {
    return this.postForm<TokenResponse>("/v1/public/api/token", req);
  }

  /** Revoke an access or refresh token. */
  async revoke(req: RevokeRequest): Promise<void> {
    await this.postFormVoid("/v1/public/api/token/revoke", req as Record<string, string | undefined>);
  }

  /** Introspect a token (requires client credentials). */
  async introspect(req: IntrospectRequest): Promise<IntrospectResponse> {
    return this.postForm<IntrospectResponse>("/v1/public/api/token/introspect", req as Record<string, string | undefined>);
  }

  // ── Session ───────────────────────────────────────────────────────────────

  /** Look up the Kratos session for the given session token. */
  async session(sessionToken: string): Promise<SessionResponse> {
    const resp = await this._fetch(`${this.baseUrl}/v1/public/api/session`, {
      headers: { Authorization: `Bearer ${sessionToken}` },
    });
    if (!resp.ok) {
      throw new IdolAuthError(resp.status, await resp.text());
    }
    return resp.json() as Promise<SessionResponse>;
  }

  // ── Headless auth ─────────────────────────────────────────────────────────

  /** Create a new account and return a session token (headless). */
  async register(req: RegisterRequest): Promise<AuthResult> {
    const resp = await this._fetch(`${this.baseUrl}/v1/public/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
    if (!resp.ok) {
      throw new IdolAuthError(resp.status, await resp.text());
    }
    return resp.json() as Promise<AuthResult>;
  }

  /** Authenticate with identifier + password and return a session token (headless). */
  async login(req: LoginRequest): Promise<AuthResult> {
    const resp = await this._fetch(`${this.baseUrl}/v1/public/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
    if (!resp.ok) {
      throw new IdolAuthError(resp.status, await resp.text());
    }
    return resp.json() as Promise<AuthResult>;
  }

  // ── Internal helpers ──────────────────────────────────────────────────────

  private async postForm<T>(
    path: string,
    params: Record<string, string | undefined>
  ): Promise<T> {
    const body = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined) body.set(k, v);
    }
    const resp = await this._fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });
    if (!resp.ok) {
      throw new IdolAuthError(resp.status, await resp.text());
    }
    return resp.json() as Promise<T>;
  }

  private async postFormVoid(
    path: string,
    params: Record<string, string | undefined>
  ): Promise<void> {
    const body = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined) body.set(k, v);
    }
    const resp = await this._fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });
    if (!resp.ok) {
      throw new IdolAuthError(resp.status, await resp.text());
    }
  }
}

export class IdolAuthError extends Error {
  constructor(
    public readonly status: number,
    message: string
  ) {
    super(`idol-auth: HTTP ${status}: ${message}`);
    this.name = "IdolAuthError";
  }
}

/** Thrown when the authorization server redirects back with an OAuth error. */
export class IdolAuthCallbackError extends Error {
  constructor(
    public readonly error: string,
    public readonly errorDescription: string
  ) {
    super(`idol-auth: authorization failed: ${error}${errorDescription ? `: ${errorDescription}` : ""}`);
    this.name = "IdolAuthCallbackError";
  }
}
