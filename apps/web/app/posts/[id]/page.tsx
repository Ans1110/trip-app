import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { PostDetail } from "@/components/posts/post-detail";

export const metadata: Metadata = {
  title: "Post — TripCraft",
};

type PageProps = { params: Promise<{ id: string }> };

export default async function PostPage({ params }: PageProps) {
  const { id } = await params;
  return (
    <PageShell back={{ href: "/feed", label: "Feed" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-3xl mx-auto">
        <PostDetail postId={id} />
      </div>
    </PageShell>
  );
}
