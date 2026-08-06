import type { Metadata } from "next";
import Link from "next/link";
import { PenSquare } from "lucide-react";

import { PageShell } from "@/components/trips/page-shell";
import { FeedSearch } from "@/components/profile/feed-search";

export const metadata: Metadata = {
  title: "Feed — TripCraft",
};

export default function FeedPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <header className="mb-8 flex items-start justify-between gap-4">
          <div>
            <p
              className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
              style={{ color: "var(--season-button)" }}
            >
              Explore
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
              Trips published by travelers you follow — or search all posts,
              tags, and authors.
            </p>
          </div>
          <Link
            href="/posts/new"
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full shrink-0"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            <PenSquare className="size-4" />
            New post
          </Link>
        </header>
        <FeedSearch />
      </div>
    </PageShell>
  );
}
