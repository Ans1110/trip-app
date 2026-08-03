import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { FeedView } from "@/components/profile/feed-view";

export const metadata: Metadata = {
  title: "Feed — TripCraft",
};

export default function FeedPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <header className="mb-8">
          <p
            className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            Following
          </p>
          <h1
            className="season-transition leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.75rem, 3vw, 2.5rem)",
              color: "var(--season-heading)",
            }}
          >
            Your feed
          </h1>
          <p className="mt-2 text-sm" style={{ color: "#8B9A8E" }}>
            Trips published by travelers you follow.
          </p>
        </header>
        <FeedView />
      </div>
    </PageShell>
  );
}
