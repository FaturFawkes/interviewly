import type { NextConfig } from "next";

const backendInternalUrl = process.env.BACKEND_INTERNAL_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
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
