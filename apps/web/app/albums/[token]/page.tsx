import type { Metadata } from "next";

import { PublicAlbumViewer } from "@/components/albums/public-album-viewer";

// robots + no-referrer keep the token out of search indexes and third-party
// Referer headers. It still appears in server access logs (path segment).
export const metadata: Metadata = {
  title: "Shared album",
  description: "A photo album shared with you.",
  robots: { index: false, follow: false, nocache: true },
  referrer: "no-referrer",
};

export default async function PublicAlbumPage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  return (
    <main
      className="min-h-screen w-full"
      style={{ backgroundColor: "#0B100D" }}
    >
      <div className="mx-auto max-w-5xl px-6 md:px-12 py-10 flex flex-col gap-6">
        <header className="flex flex-col gap-1">
          <p
            className="text-[11px] font-medium tracking-[0.2em] uppercase"
            style={{ color: "var(--season-button)" }}
          >
            Shared album
          </p>
          <h1
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            A moment from the trip
          </h1>
        </header>
        <PublicAlbumViewer token={token} />
      </div>
    </main>
  );
}
