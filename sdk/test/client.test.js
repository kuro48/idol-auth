const assert = require("node:assert/strict");
const test = require("node:test");

const { IdolAuthClient, IdolAuthError } = require("../dist");

function jsonResponse(data, init = {}) {
  return new Response(JSON.stringify(data), {
    status: init.status || 200,
    headers: { "content-type": "application/json" },
  });
}

test("browser helper URLs are encoded and use the public endpoints", () => {
  const client = new IdolAuthClient({ baseUrl: "https://auth.example.com/" });

  assert.equal(
    client.browserLoginUrl({
      clientId: "cid",
      redirectUri: "https://app.example.com/cb",
      scope: "openid profile",
      state: "s",
      nonce: "n",
      codeChallenge: "cc",
      codeChallengeMethod: "S256",
    }),
    "https://auth.example.com/v1/public/browser/login?client_id=cid&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcb&response_type=code&scope=openid+profile&state=s&nonce=n&code_challenge=cc&code_challenge_method=S256"
  );

  assert.equal(
    client.browserRegistrationUrl("https://app.example.com/welcome?a=1&b=2"),
    "https://auth.example.com/v1/public/browser/registration?return_to=https%3A%2F%2Fapp.example.com%2Fwelcome%3Fa%3D1%26b%3D2"
  );

  assert.equal(
    client.browserLogoutUrl({
      idTokenHint: "id",
      postLogoutRedirectUri: "https://app.example.com/out",
      state: "bye",
    }),
    "https://auth.example.com/v1/public/browser/logout?id_token_hint=id&post_logout_redirect_uri=https%3A%2F%2Fapp.example.com%2Fout&state=bye"
  );
});

test("token operations send form-encoded requests", async () => {
  const calls = [];
  const client = new IdolAuthClient({
    baseUrl: "https://auth.example.com/",
    fetch: async (url, init = {}) => {
      calls.push({ url, init });
      return jsonResponse({
        access_token: "at",
        token_type: "bearer",
        expires_in: 300,
        active: true,
      });
    },
  });

  await client.token({
    grant_type: "authorization_code",
    code: "code",
    redirect_uri: "https://app.example.com/cb",
    client_id: "cid",
    code_verifier: "verifier",
  });
  assert.equal(calls.at(-1).url, "https://auth.example.com/v1/public/api/token");
  assert.equal(calls.at(-1).init.method, "POST");
  assert.equal(
    calls.at(-1).init.headers["Content-Type"],
    "application/x-www-form-urlencoded"
  );
  assert.equal(
    calls.at(-1).init.body,
    "grant_type=authorization_code&code=code&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcb&client_id=cid&code_verifier=verifier"
  );

  await client.introspect({
    token: "at",
    client_id: "cid",
    client_secret: "secret",
  });
  assert.equal(
    calls.at(-1).url,
    "https://auth.example.com/v1/public/api/token/introspect"
  );
  assert.equal(calls.at(-1).init.body, "token=at&client_id=cid&client_secret=secret");
});

test("revoke accepts successful empty responses", async () => {
  let call;
  const client = new IdolAuthClient({
    baseUrl: "https://auth.example.com",
    fetch: async (url, init = {}) => {
      call = { url, init };
      return new Response("", { status: 200 });
    },
  });

  await client.revoke({ token: "refresh-token", client_id: "cid" });

  assert.equal(call.url, "https://auth.example.com/v1/public/api/token/revoke");
  assert.equal(call.init.method, "POST");
  assert.equal(call.init.body, "token=refresh-token&client_id=cid");
});

test("session and headless auth use the expected endpoints", async () => {
  const calls = [];
  const client = new IdolAuthClient({
    baseUrl: "https://auth.example.com",
    fetch: async (url, init = {}) => {
      calls.push({ url, init });
      return jsonResponse({
        active: true,
        session_token: "st",
        identity_id: "id",
        email: "a@example.com",
      });
    },
  });

  await client.session("session-token");
  assert.equal(calls.at(-1).url, "https://auth.example.com/v1/public/api/session");
  assert.equal(calls.at(-1).init.headers.Authorization, "Bearer session-token");

  await client.register({
    email: "a@example.com",
    password: "pw",
    display_name: "A",
  });
  assert.equal(calls.at(-1).url, "https://auth.example.com/v1/public/api/register");
  assert.equal(calls.at(-1).init.headers["Content-Type"], "application/json");
  assert.equal(
    calls.at(-1).init.body,
    JSON.stringify({ email: "a@example.com", password: "pw", display_name: "A" })
  );

  await client.login({ identifier: "a@example.com", password: "pw" });
  assert.equal(calls.at(-1).url, "https://auth.example.com/v1/public/api/login");
});

test("non-2xx responses throw IdolAuthError", async () => {
  const client = new IdolAuthClient({
    baseUrl: "https://auth.example.com",
    fetch: async () => new Response("bad credentials", { status: 401 }),
  });

  await assert.rejects(
    () => client.login({ identifier: "x", password: "bad" }),
    (err) =>
      err instanceof IdolAuthError &&
      err.status === 401 &&
      err.message.includes("bad credentials")
  );
});
