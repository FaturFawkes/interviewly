import type { NextConfig } from "next";

const backendInternalUrl = process.env.BACKEND_INTERNAL_URL ?? "http://localhost:8080";
const appEnv = (process.env.APP_ENV ?? "").trim();

const nextConfig: NextConfig = {
  env: {
    // Reuse backend APP_ENV as frontend runtime environment signal.
    NEXT_PUBLIC_APP_ENV: appEnv,
  },
  async rewrites() {
    return [
      {
        source: "/api-proxy/:path*",
        destination: `${backendInternalUrl}/:path*`,
      },
    ];
  },
};

export default nextConfig;
