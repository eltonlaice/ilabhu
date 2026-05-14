import type { NextConfig } from "next";

const apiBase = process.env.ILABHU_API_BASE ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Standalone output: the production build emits a self-contained server.js
  // and a minimal node_modules under .next/standalone/ that the Dockerfile
  // copies into a slim runtime image. See deploy/docker-compose.yml.
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiBase}/:path*`,
      },
    ];
  },
};

export default nextConfig;
