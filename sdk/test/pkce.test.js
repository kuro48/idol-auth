const assert = require("node:assert/strict");
const test = require("node:test");

const {
  IdolAuthClient,
  generateCodeVerifier,
  generateCodeChallenge,
  generatePKCE,
} = require("../dist/cjs/index.js");

function fakeStorage() {
  const store = new Map();
  return {
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, v),
    removeItem: (k) => store.delete(k),
  };
}

function jsonResponse(data) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

test("generateCodeVerifier produces RFC 7636 compliant verifiers", () => {
  const seen = new Set();
  for (let i = 0; i < 50; i++) {
    const v = generateCodeVerifier();
    assert.ok(v.length >= 43 && v.length <= 128, `length ${v.length} out of range`);
    assert.match(v, /^[A-Za-z0-9\-._~]+$/);
    seen.add(v);
  }
  assert.equal(seen.size, 50, "verifiers must be unique");
});

test("generateCodeChallenge matches the RFC 7636 appendix B vector", async () => {
  const challenge = await generateCodeChallenge(
    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
  );
  assert.equal(challenge, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM");
});

test("generatePKCE returns a matching pair", async () => {
  const pkce = await generatePKCE();
  assert.equal(pkce.codeChallengeMethod, "S256");
  assert.equal(await generateCodeChallenge(pkce.codeVerifier), pkce.codeChallenge);
});

test("loginWithRedirect stores the transaction and builds the authorize URL", async () => {
  const client = new IdolAuthClient({ baseUrl: "https://auth.example.com" });
  const storage = fakeStorage();
  let navigatedTo = "";

  const url = await client.loginWithRedirect({
    clientId: "cid",
    redirectUri: "https://app.example.com/cb",
    storage,
    navigate: (u) => {
      navigatedTo = u;
    },
  });

  assert.equal(navigatedTo, url);
  const parsed = new URL(url);
  assert.equal(parsed.pathname, "/v1/public/browser/login");
  assert.equal(parsed.searchParams.get("client_id"), "cid");
  assert.equal(parsed.searchParams.get("code_challenge_method"), "S256");
  assert.equal(
    parsed.searchParams.get("scope"),
    "openid email profile offline_access"
  );

  const tx = JSON.parse(storage.getItem("idol_auth_tx"));
  assert.equal(tx.clientId, "cid");
  assert.equal(tx.state, parsed.searchParams.get("state"));
  assert.equal(
    await generateCodeChallenge(tx.codeVerifier),
    parsed.searchParams.get("code_challenge")
  );
});

test("handleRedirectCallback exchanges the code with the stored verifier", async () => {
  const calls = [];
  const client = new IdolAuthClient({
    baseUrl: "https://auth.example.com",
    fetch: async (url, init) => {
      calls.push({ url, init });
      return jsonResponse({ access_token: "at", token_type: "bearer", expires_in: 300 });
    },
  });
  const storage = fakeStorage();

  const loginUrl = await client.loginWithRedirect({
    clientId: "cid",
    redirectUri: "https://app.example.com/cb",
    storage,
    navigate: () => {},
  });
  const state = new URL(loginUrl).searchParams.get("state");
  const verifier = JSON.parse(storage.getItem("idol_auth_tx")).codeVerifier;

  const tokens = await client.handleRedirectCallback({
    url: `https://app.example.com/cb?code=abc123&state=${state}`,
    storage,
  });

  assert.equal(tokens.access_token, "at");
  assert.equal(calls.length, 1);
  const body = new URLSearchParams(calls[0].init.body);
  assert.equal(body.get("grant_type"), "authorization_code");
  assert.equal(body.get("code"), "abc123");
  assert.equal(body.get("code_verifier"), verifier);
  assert.equal(body.get("client_id"), "cid");
  assert.equal(body.get("redirect_uri"), "https://app.example.com/cb");
  assert.equal(storage.getItem("idol_auth_tx"), null, "transaction must be cleared");
});

test("handleRedirectCallback rejects a state mismatch", async () => {
  const client = new IdolAuthClient({ baseUrl: "https://auth.example.com" });
  const storage = fakeStorage();

  await client.loginWithRedirect({
    clientId: "cid",
    redirectUri: "https://app.example.com/cb",
    storage,
    navigate: () => {},
  });

  await assert.rejects(
    client.handleRedirectCallback({
      url: "https://app.example.com/cb?code=abc&state=tampered",
      storage,
    }),
    /state mismatch/
  );
  assert.equal(storage.getItem("idol_auth_tx"), null);
});

test("handleRedirectCallback surfaces OAuth errors from the callback", async () => {
  const client = new IdolAuthClient({ baseUrl: "https://auth.example.com" });
  const storage = fakeStorage();
  storage.setItem("idol_auth_tx", JSON.stringify({ state: "s" }));

  await assert.rejects(
    client.handleRedirectCallback({
      url: "https://app.example.com/cb?error=access_denied&error_description=User+cancelled",
      storage,
    }),
    (err) => {
      assert.equal(err.name, "IdolAuthCallbackError");
      assert.equal(err.error, "access_denied");
      return true;
    }
  );
});
