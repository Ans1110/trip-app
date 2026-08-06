import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { EditPostView } from "@/components/posts/edit-post-view";

export const metadata: Metadata = {
  title: "Edit post — TripCraft",
};

type PageProps = { params: Promise<{ id: string }> };

export default async function EditPostPage({ params }: PageProps) {
  const { id } = await params;
  return (
    <PageShell back={{ href: `/posts/${id}`, label: "Back to post" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <header className="mb-8">
          <p
            className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            Edit
          </p>
          <h1
            className="season-transition leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.75rem, 3vw, 2.5rem)",
              color: "var(--season-heading)",
            }}
          >
            Update your post
          </h1>
        </header>
        <EditPostView postId={id} />
      </div>
    </PageShell>
  );
}
