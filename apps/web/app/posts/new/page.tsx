import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { PostForm } from "@/components/posts/post-form";

export const metadata: Metadata = {
  title: "New post — TripCraft",
};

export default function NewPostPage() {
  return (
    <PageShell back={{ href: "/feed", label: "Feed" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <header className="mb-8">
          <p
            className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            New post
          </p>
          <h1
            className="season-transition leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.75rem, 3vw, 2.5rem)",
              color: "var(--season-heading)",
            }}
          >
            Share a story
          </h1>
        </header>
        <PostForm mode="create" />
      </div>
    </PageShell>
  );
}
