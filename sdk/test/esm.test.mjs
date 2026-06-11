import assert from "node:assert/strict";
import test from "node:test";

import { IdolAuthClient, generatePKCE } from "../dist/esm/index.js";

test("ESM build exports the client and PKCE helpers", async () => {
  const client = new IdolAuthClient({ baseUrl: "https://auth.example.com" });
  assert.equal(typeof client.loginWithRedirect, "function");
  assert.equal(typeof client.handleRedirectCallback, "function");

  const pkce = await generatePKCE();
  assert.equal(pkce.codeChallengeMethod, "S256");
});
