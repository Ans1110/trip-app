import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    formats: ["image/avif", "image/webp"],
    qualities: [75, 95],
    minimumCacheTTL: 60 * 60 * 24 * 30,
  },
};

export default nextConfig;
