import type { Metadata } from "next";
import { Suspense } from "react";
import Link from "next/link";
import { Compass } from "lucide-react";

import { GithubOAuthCallback } from "@/components/auth/github-oauth-callback";

export const metadata: Metadata = {
  title: "Continuing with GitHub — TripCraft",
};

export default function GithubOAuthCallbackPage() {
  return (
    <main
      className="season-transition relative min-h-screen flex flex-col"
      style={{
        backgroundColor: "#0B100D",
        color: "#ECEFEA",
        backgroundImage:
          "radial-gradient(circle 600px at 80% -10%, color-mix(in srgb, var(--season-button) 22%, transparent), transparent 70%), radial-gradient(circle 500px at 10% 110%, color-mix(in srgb, var(--season-accent) 14%, transparent), transparent 70%)",
      }}
    >
      <header className="px-6 md:px-12 h-16 flex items-center">
        <Link href="/" className="flex items-center gap-2 select-none">
          <Compass
            className="season-transition size-5"
            style={{ color: "var(--season-button)" }}
          />
          <span
            className="text-xl tracking-tight"
            style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
          >
            TripCraft
          </span>
        </Link>
      </header>

      <section className="flex-1 flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8">
            <h1
              className="season-transition leading-[1.05] tracking-tight mb-3"
              style={{
                fontFamily: "var(--font-display, Georgia, serif)",
                fontSize: "clamp(2rem, 4vw, 2.75rem)",
                color: "var(--season-heading)",
              }}
            >
              Almost there.
            </h1>
            <p className="text-sm" style={{ color: "#8B9A8E" }}>
              Finishing your GitHub sign-in.
            </p>
          </div>

          <Suspense fallback={null}>
            <GithubOAuthCallback />
          </Suspense>
        </div>
      </section>
    </main>
  );
}
