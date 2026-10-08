const goAPIProxy =
  process.env.GO_API_PROXY ||
  (process.env.NODE_ENV === "development" && !process.env.VERCEL
    ? "http://localhost:8080"
    : "");

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    if (!goAPIProxy) return [];
    return [
      { source: "/api/:path*", destination: `${goAPIProxy}/api/:path*` },
      { source: "/auth/callback", destination: `${goAPIProxy}/auth/callback` },
    ];
  },
};

export default nextConfig;
