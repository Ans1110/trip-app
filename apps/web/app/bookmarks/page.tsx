import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { BookmarksView } from "@/components/profile/bookmarks-view";

export const metadata: Metadata = {
  title: "Bookmarks — TripCraft",
};

export default function BookmarksPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <header className="mb-8">
          <p
            className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            Saved
          </p>
          <h1
            className="season-transition leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.75rem, 3vw, 2.5rem)",
              color: "var(--season-heading)",
            }}
          >
            Bookmarks
          </h1>
          <p className="mt-2 text-sm" style={{ color: "#8B9A8E" }}>
            Posts you&rsquo;ve saved. Only you can see this list.
          </p>
        </header>
        <BookmarksView />
      </div>
    </PageShell>
  );
}
