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
} from "./types";

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

  // ── Token operations ──────────────────────────────────────────────────────

  /** Exchange an authorization code (or refresh token) for tokens. */
  async token(req: TokenRequest): Promise<TokenResponse> {
    return this.postForm<TokenResponse>("/v1/public/api/token", req);
  }

  /** Revoke an access or refresh token. */
  async revoke(req: RevokeRequest): Promise<void> {
    await this.postForm<unknown>("/v1/public/api/token/revoke", req as Record<string, string | undefined>);
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
