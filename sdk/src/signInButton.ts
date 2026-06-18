import type { IdolAuthClient } from "./client.js";
import type { LoginWithRedirectOptions } from "./types.js";

/** OIDC social provider identifier passed to Kratos login flow. */
export type SocialProvider = "google" | "twitter" | "apple";

export interface SignInWithIdoliumOptions {
  /** Configured IdolAuthClient instance. */
  client: IdolAuthClient;
  /** OAuth2 client_id of the consuming application. */
  clientId: string;
  /** Registered redirect URI to receive the authorization code. */
  redirectUri: string;
  /** Override the default scope (`openid email profile offline_access`). */
  scope?: string;
  /** Button label. Defaults to "idoliumでログイン". */
  label?: string;
  /** Color scheme. `auto` follows `prefers-color-scheme`. */
  theme?: "light" | "dark" | "auto";
  /** Disable the button. */
  disabled?: boolean;
  /**
   * Optional social/OIDC provider to use for login.
   * When set the Kratos login URL will include `?provider=<id>` so the browser
   * is taken directly to the social provider's OAuth screen.
   */
  provider?: SocialProvider;
  /** Called before the redirect is initiated. */
  onClick?: (event: MouseEvent) => void;
  /** Called if loginWithRedirect throws. */
  onError?: (error: unknown) => void;
  /** Optional storage override forwarded to loginWithRedirect. */
  storage?: LoginWithRedirectOptions["storage"];
  /** Optional navigate override forwarded to loginWithRedirect. */
  navigate?: LoginWithRedirectOptions["navigate"];
}

const STYLE_ID = "idol-auth-signin-style";

const BUTTON_CSS = `
.idol-auth-signin-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 18px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  font-size: 15px;
  font-weight: 600;
  line-height: 1;
  border-radius: 8px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, transform 80ms ease;
  user-select: none;
}
.idol-auth-signin-btn:focus-visible {
  outline: 2px solid #4f7cff;
  outline-offset: 2px;
}
.idol-auth-signin-btn:active:not(:disabled) {
  transform: translateY(1px);
}
.idol-auth-signin-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.idol-auth-signin-btn[data-theme="light"] {
  background: #ffffff;
  color: #1f2937;
  border-color: #d1d5db;
}
.idol-auth-signin-btn[data-theme="light"]:hover:not(:disabled) {
  background: #f9fafb;
}
.idol-auth-signin-btn[data-theme="dark"] {
  background: #1f2937;
  color: #f9fafb;
  border-color: #1f2937;
}
.idol-auth-signin-btn[data-theme="dark"]:hover:not(:disabled) {
  background: #111827;
}
.idol-auth-signin-btn[data-theme="auto"] {
  background: #ffffff;
  color: #1f2937;
  border-color: #d1d5db;
}
@media (prefers-color-scheme: dark) {
  .idol-auth-signin-btn[data-theme="auto"] {
    background: #1f2937;
    color: #f9fafb;
    border-color: #1f2937;
  }
  .idol-auth-signin-btn[data-theme="auto"]:hover:not(:disabled) {
    background: #111827;
  }
}
.idol-auth-signin-btn__icon {
  display: inline-block;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: linear-gradient(135deg, #b2d8ff 0%, #d8b2ff 50%, #ffb2d8 100%);
  flex-shrink: 0;
}
`;

function ensureStyles(doc: Document): void {
  if (doc.getElementById(STYLE_ID)) return;
  const style = doc.createElement("style");
  style.id = STYLE_ID;
  style.textContent = BUTTON_CSS;
  doc.head.appendChild(style);
}

/**
 * Build a "idoliumでログイン" button that kicks off the OAuth2 authorization
 * code + PKCE flow when clicked. The returned element can be appended into
 * any DOM container.
 */
export function createSignInWithIdoliumButton(
  options: SignInWithIdoliumOptions
): HTMLButtonElement {
  if (typeof document === "undefined") {
    throw new Error(
      "createSignInWithIdoliumButton requires a browser environment"
    );
  }
  ensureStyles(document);

  const button = document.createElement("button");
  button.type = "button";
  button.className = "idol-auth-signin-btn";
  button.dataset.theme = options.theme ?? "auto";
  if (options.disabled) button.disabled = true;

  const icon = document.createElement("span");
  icon.className = "idol-auth-signin-btn__icon";
  icon.setAttribute("aria-hidden", "true");

  const label = document.createElement("span");
  label.textContent = options.label ?? "idoliumでログイン";

  button.appendChild(icon);
  button.appendChild(label);

  button.addEventListener("click", async event => {
    if (options.onClick) options.onClick(event);
    if (event.defaultPrevented) return;
    button.disabled = true;
    try {
      await options.client.loginWithRedirect({
        clientId: options.clientId,
        redirectUri: options.redirectUri,
        scope: options.scope,
        storage: options.storage,
        navigate: options.navigate,
      });
    } catch (err) {
      button.disabled = false;
      if (options.onError) {
        options.onError(err);
      } else {
        throw err;
      }
    }
  });

  return button;
}

/**
 * Convenience helper that creates the button and appends it to the given
 * container. Returns the button for further customization.
 */
export function mountSignInWithIdoliumButton(
  container: HTMLElement,
  options: SignInWithIdoliumOptions
): HTMLButtonElement {
  const button = createSignInWithIdoliumButton(options);
  container.appendChild(button);
  return button;
}
