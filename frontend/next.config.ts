import type { NextConfig } from "next";
import { PROJECTS } from "./lib/projects";

const backendURL = process.env.BACKEND_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  reactCompiler: true,
  // TypeScript 7 ships a native compiler without the classic JS API that
  // Next's default typecheck uses — run project-local `tsc` instead.
  experimental: {
    useTypeScriptCli: true,
  },
  async redirects() {
    // Keep shared download pages and version-pinned links working after transfer.
    return PROJECTS.flatMap(({ owner, repo }) =>
      ["", "/p", "/api/release", "/api/releases", "/api/readme", "/api/link", "/dl", "/code"].map((prefix) => ({
        source: `${prefix}/Xinrea/${repo}/:path*`,
        destination: `${prefix}/${owner}/${repo}/:path*`,
        permanent: true,
      })),
    );
  },
  async rewrites() {
    // Local/dev proxy to the Go backend. On Vercel, root vercel.json routes
    // these paths to the backend service before Next.js sees the request.
    return {
      beforeFiles: [
        { source: "/dl/:path*", destination: `${backendURL}/dl/:path*` },
        { source: "/code/:path*", destination: `${backendURL}/code/:path*` },
        { source: "/health", destination: `${backendURL}/health` },
      ],
      afterFiles: [
        { source: "/api/:path*", destination: `${backendURL}/api/:path*` },
        // Public project links transparently serve the matching /p/ release
        // page while keeping the short /:owner/:repo URL in the address bar.
        // OG/Twitter image routes stay off /p/ so robots Disallow:/p/ does not
        // block unfurl image fetches for scrapers that honor the * rule.
        {
          source: "/:owner/:repo/opengraph-image",
          destination: "/p/:owner/:repo/opengraph-image",
        },
        {
          source: "/:owner/:repo/twitter-image",
          destination: "/p/:owner/:repo/twitter-image",
        },
        { source: "/:owner/:repo", destination: "/p/:owner/:repo" },
      ],
      fallback: [],
    };
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "github.com",
        pathname: "/*.png",
      },
      {
        // github.com/<owner>.png 302s here - Next won't follow a redirect
        // to a host that isn't itself allow-listed.
        protocol: "https",
        hostname: "avatars.githubusercontent.com",
      },
    ],
  },
};

export default nextConfig;
