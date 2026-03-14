import type { NextAuthOptions } from "next-auth";
import GoogleProvider from "next-auth/providers/google";
import AzureADProvider from "next-auth/providers/azure-ad";
import CredentialsProvider from "next-auth/providers/credentials";

const backendAuthBaseUrl =
  process.env.BACKEND_AUTH_BASE_URL ?? process.env.BACKEND_INTERNAL_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

function getBackendAuthUrl(): string {
  if (backendAuthBaseUrl.startsWith("http://") || backendAuthBaseUrl.startsWith("https://")) {
    return backendAuthBaseUrl;
  }

  const appBaseUrl = process.env.NEXTAUTH_URL ?? "http://localhost:3000";
  return `${appBaseUrl}${backendAuthBaseUrl}`;
}

type BackendAuthResponse = {
  access_token: string;
  user: {
    id: string;
    email: string;
    full_name?: string;
  };
};

async function exchangeSocialLogin(payload: {
  provider: "google" | "microsoft";
  providerAccountID: string;
  email: string;
  fullName: string;
}): Promise<BackendAuthResponse> {
  const response = await fetch(`${getBackendAuthUrl()}/auth/social-login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      provider: payload.provider,
      provider_account_id: payload.providerAccountID,
      email: payload.email,
      full_name: payload.fullName,
    }),
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error("failed to exchange social login token");
  }

  return (await response.json()) as BackendAuthResponse;
}

async function exchangePasswordAuth(payload: {
  email: string;
  password: string;
}): Promise<BackendAuthResponse> {
  const response = await fetch(`${getBackendAuthUrl()}/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      email: payload.email,
      password: payload.password,
    }),
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error("invalid credentials or registration failed");
  }

  return (await response.json()) as BackendAuthResponse;
}

export const authOptions: NextAuthOptions = {
  providers: [
    GoogleProvider({
      clientId: process.env.AUTH_GOOGLE_ID ?? "",
      clientSecret: process.env.AUTH_GOOGLE_SECRET ?? "",
    }),
    AzureADProvider({
      clientId: process.env.AUTH_AZURE_AD_ID ?? "",
      clientSecret: process.env.AUTH_AZURE_AD_SECRET ?? "",
      tenantId: process.env.AUTH_AZURE_AD_TENANT_ID,
    }),
    CredentialsProvider({
      name: "Email & Password",
      credentials: {
        email: { label: "Email", type: "email" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        const email = typeof credentials?.email === "string" ? credentials.email.trim() : "";
        const password = typeof credentials?.password === "string" ? credentials.password : "";

        if (!email || !password) {
          return null;
        }

        try {
          const result = await exchangePasswordAuth({
            email,
            password,
          });

          return {
            id: result.user.id,
            email: result.user.email,
            name: result.user.full_name,
            backendAccessToken: result.access_token,
          };
        } catch {
          return null;
        }
      },
    }),
  ],
  session: {
    strategy: "jwt",
  },
  callbacks: {
    async jwt({ token, account, profile, user }) {
      if (account?.provider === "credentials") {
        const credentialUser = user as { backendAccessToken?: string; id?: string } | undefined;
        if (typeof credentialUser?.backendAccessToken === "string") {
          token.backendAccessToken = credentialUser.backendAccessToken;
        }
        if (typeof credentialUser?.id === "string") {
          token.userId = credentialUser.id;
        }
      }

      if (account?.provider === "google" || account?.provider === "azure-ad") {
        const provider = account.provider === "google" ? "google" : "microsoft";
        const email = typeof token.email === "string" ? token.email : "";
        const profileName =
          profile && typeof profile === "object" && "name" in profile && typeof profile.name === "string" ? profile.name : "";
        const fullName = (typeof token.name === "string" && token.name) || profileName;
        const providerAccountID = account.providerAccountId;

        if (email && providerAccountID) {
          try {
            const exchanged = await exchangeSocialLogin({
              provider,
              providerAccountID,
              email,
              fullName,
            });

            token.backendAccessToken = exchanged.access_token;
            token.userId = exchanged.user.id;
            delete token.authError;
          } catch {
            token.authError = "Failed to sign in with backend";
          }
        }
      }

      return token;
    },
    async session({ session, token }) {
      if (typeof token.backendAccessToken === "string") {
        session.backendAccessToken = token.backendAccessToken;
      }

      if (typeof token.userId === "string" && session.user) {
        session.user.id = token.userId;
      }

      if (typeof token.authError === "string") {
        session.authError = token.authError;
      }

      return session;
    },
  },
  pages: {
    signIn: "/auth/sign-in",
  },
};
