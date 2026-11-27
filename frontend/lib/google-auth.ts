export const GOOGLE_CLIENT_ID = (
  process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? ""
).trim();

const GOOGLE_GSI_SCRIPT_SRC = "https://accounts.google.com/gsi/client";
const GOOGLE_GSI_LOAD_TIMEOUT_MS = 10_000;

export interface GoogleCredentialResponse {
  credential?: string;
  select_by?: string;
  clientId?: string;
}

export interface GoogleIdConfiguration {
  client_id: string;
  callback: (response: GoogleCredentialResponse) => void;
  nonce?: string;
  ux_mode?: "popup" | "redirect";
  context?: "signin" | "signup" | "use";
  auto_select?: boolean;
  cancel_on_tap_outside?: boolean;
  itp_support?: boolean;
}

export interface GoogleRenderButtonConfiguration {
  type?: "standard" | "icon";
  theme?: "outline" | "filled_blue" | "filled_black";
  size?: "large" | "medium" | "small";
  text?: "signin_with" | "signup_with" | "continue_with" | "signin";
  shape?: "rectangular" | "pill" | "circle" | "square";
  logo_alignment?: "left" | "center";
  width?: number;
  locale?: string;
}

export interface GoogleAccountsId {
  initialize: (config: GoogleIdConfiguration) => void;
  renderButton: (
    parent: HTMLElement,
    options: GoogleRenderButtonConfiguration
  ) => void;
  cancel: () => void;
}

declare global {
  interface Window {
    google?: {
      accounts?: {
        id?: GoogleAccountsId;
      };
    };
  }
}

function bytesToLowerHex(bytes: Uint8Array): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export async function generateGoogleNonce(): Promise<{
  rawNonce: string;
  hashedNonce: string;
}> {
  if (
    typeof window === "undefined" ||
    !window.crypto?.getRandomValues ||
    !window.crypto?.subtle?.digest
  ) {
    throw new Error("Web Crypto API is unavailable");
  }

  const randomBytes = new Uint8Array(32);
  window.crypto.getRandomValues(randomBytes);
  const rawNonce = bytesToLowerHex(randomBytes);

  const encoded = new TextEncoder().encode(rawNonce);
  const digest = await window.crypto.subtle.digest("SHA-256", encoded);
  const hashedNonce = bytesToLowerHex(new Uint8Array(digest));

  return { rawNonce, hashedNonce };
}

let gsiScriptPromise: Promise<void> | null = null;

export function loadGoogleGsiScript(): Promise<void> {
  if (typeof window === "undefined" || typeof document === "undefined") {
    return Promise.reject(new Error("Browser environment is required"));
  }

  if (window.google?.accounts?.id) {
    return Promise.resolve();
  }

  if (gsiScriptPromise) {
    return gsiScriptPromise;
  }

  gsiScriptPromise = new Promise<void>((resolve, reject) => {
    const existingScript = document.querySelector<HTMLScriptElement>(
      `script[src="${GOOGLE_GSI_SCRIPT_SRC}"]`
    );
    const createdScript = !existingScript;
    const script = existingScript ?? document.createElement("script");

    let settled = false;

    const cleanup = () => {
      window.clearTimeout(timeoutId);
      script.removeEventListener("load", handleLoad);
      script.removeEventListener("error", handleError);
    };

    const fail = (message: string) => {
      if (settled) return;
      settled = true;
      cleanup();
      gsiScriptPromise = null;
      if (createdScript && script.parentNode) {
        script.parentNode.removeChild(script);
      }
      reject(new Error(message));
    };

    const succeed = () => {
      if (settled) return;
      if (!window.google?.accounts?.id) {
        fail("Google Identity Services loaded without accounts.id");
        return;
      }
      settled = true;
      cleanup();
      resolve();
    };

    const handleLoad = () => {
      succeed();
    };

    const handleError = () => {
      fail("Failed to load Google Identity Services script");
    };

    const timeoutId = window.setTimeout(() => {
      fail("Timed out loading Google Identity Services script");
    }, GOOGLE_GSI_LOAD_TIMEOUT_MS);

    script.addEventListener("load", handleLoad, { once: true });
    script.addEventListener("error", handleError, { once: true });

    if (createdScript) {
      script.src = GOOGLE_GSI_SCRIPT_SRC;
      script.async = true;
      script.defer = true;
      document.head.appendChild(script);
    } else if (window.google?.accounts?.id) {
      succeed();
    }
  });

  return gsiScriptPromise;
}
