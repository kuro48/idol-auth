# @idol-auth/client

TypeScript client for the [idol-auth](https://github.com/kuro48/idol-auth) public API. PKCE helpers built in — no extra dependencies, works in browsers and Node.js 18+.

## Install

```bash
npm install @idol-auth/client
```

## Quickstart (SPA / Web)

```ts
import { IdolAuthClient } from "@idol-auth/client";

const client = new IdolAuthClient({ baseUrl: "https://auth.example.com" });

// 1. Login button — PKCE and state are handled for you.
await client.loginWithRedirect({
  clientId: "your-client-id",
  redirectUri: "https://yourapp.example.com/callback",
});

// 2. Callback page — exchanges the code for tokens.
const tokens = await client.handleRedirectCallback();
// tokens.access_token, tokens.id_token, tokens.refresh_token
```

That's it. `loginWithRedirect` generates the PKCE verifier/challenge and state, keeps them in `sessionStorage`, and redirects to the authorization page. `handleRedirectCallback` validates state and completes the token exchange.

### Refresh

```ts
const refreshed = await client.token({
  grant_type: "refresh_token",
  refresh_token: tokens.refresh_token!,
  client_id: "your-client-id",
});
```

### Logout

```ts
window.location.href = client.browserLogoutUrl({
  idTokenHint: tokens.id_token,
  postLogoutRedirectUri: "https://yourapp.example.com/",
});
```

## Headless auth (mobile / backend)

```ts
const result = await client.login({
  identifier: "user@example.com",
  password: "secret",
});
// result.session_token

const session = await client.session(result.session_token);
// session.active, session.email, session.roles
```

## Low-level PKCE helpers

```ts
import { generatePKCE, generateCodeVerifier, generateCodeChallenge } from "@idol-auth/client";

const { codeVerifier, codeChallenge } = await generatePKCE();
```

## Getting a client ID

Sign in to your idol-auth account and register an app at `/developer/app-requests/new` — an OIDC client and credentials are issued instantly.

## License

MIT
