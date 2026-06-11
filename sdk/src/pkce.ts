// PKCE (RFC 7636) helpers built on the Web Crypto API.
// Works in browsers and Node.js 18+ without extra dependencies.

export interface PKCEPair {
  codeVerifier: string;
  codeChallenge: string;
  codeChallengeMethod: "S256";
}

function getCrypto(): Crypto {
  const c = globalThis.crypto;
  if (!c || !c.subtle) {
    throw new Error(
      "idol-auth: Web Crypto API is not available in this environment (requires a modern browser or Node.js 18+)"
    );
  }
  return c;
}

function base64url(bytes: Uint8Array): string {
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** Returns a cryptographically random 64-character code verifier. */
export function generateCodeVerifier(): string {
  // 48 random bytes -> 64 base64url chars, within RFC 7636's 43-128 range.
  const bytes = new Uint8Array(48);
  getCrypto().getRandomValues(bytes);
  return base64url(bytes);
}

/** Computes the S256 code challenge for a verifier. */
export async function generateCodeChallenge(codeVerifier: string): Promise<string> {
  const digest = await getCrypto().subtle.digest(
    "SHA-256",
    new TextEncoder().encode(codeVerifier)
  );
  return base64url(new Uint8Array(digest));
}

/** Generates a matching verifier/challenge pair for the S256 method. */
export async function generatePKCE(): Promise<PKCEPair> {
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallenge(codeVerifier);
  return { codeVerifier, codeChallenge, codeChallengeMethod: "S256" };
}
