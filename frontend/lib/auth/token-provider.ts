export type TokenProvider = () => string | null | Promise<string | null>;

let tokenProvider: TokenProvider = () => null;
const TOKEN_STORAGE_KEY = "ai_interview_token";
let cachedSessionToken: string | null = null;
let cachedSessionTokenAt = 0;

export function setTokenProvider(provider: TokenProvider): void {
  tokenProvider = provider;
}

export async function getAuthToken(): Promise<string | null> {
  const provided = await tokenProvider();
  if (provided) {
    return provided;
  }

  if (typeof window !== "undefined") {
    const now = Date.now();
    if (cachedSessionToken && now-cachedSessionTokenAt < 30000) {
      return cachedSessionToken;
    }

    try {
      const response = await fetch("/api/auth/backend-token", {
        method: "GET",
        credentials: "include",
        cache: "no-store",
      });

      if (response.ok) {
        const payload = (await response.json()) as { access_token?: string };
        if (payload.access_token) {
          cachedSessionToken = payload.access_token;
          cachedSessionTokenAt = now;
          return payload.access_token;
        }
      }
    } catch {
      cachedSessionToken = null;
      cachedSessionTokenAt = 0;
    }
  }

  if (typeof window !== "undefined") {
    const stored = window.localStorage.getItem(TOKEN_STORAGE_KEY);
    if (stored) {
      return stored;
    }
  }

  const devToken = process.env.NEXT_PUBLIC_DEV_JWT_TOKEN;
  if (devToken && devToken.trim() !== "") {
    return devToken;
  }

  return null;
}

export function setStoredAuthToken(token: string | null): void {
  if (typeof window === "undefined") {
    return;
  }

  if (!token) {
    window.localStorage.removeItem(TOKEN_STORAGE_KEY);
    cachedSessionToken = null;
    cachedSessionTokenAt = 0;
    return;
  }

  window.localStorage.setItem(TOKEN_STORAGE_KEY, token);
  cachedSessionToken = token;
  cachedSessionTokenAt = Date.now();
}
