"use client";

import { ImageIcon, Link2Off } from "lucide-react";

import { AlbumPhotoCard } from "@/components/trips/album/album-photo-card";
import {
  classifyPublicAlbumError,
  useAlbumPublic,
  type PublicAlbumErrorKind,
} from "@/hooks/album-hooks";
import type { AlbumPhoto } from "@/lib/apis/album-api";

export function PublicAlbumViewer({ token }: { token: string }) {
  const q = useAlbumPublic(token);

  if (q.isLoading) {
    return <Message tone="muted">Loading album…</Message>;
  }

  if (q.isError) {
    return <PublicAlbumError kind={classifyPublicAlbumError(q.error)} />;
  }

  const photos = q.data?.photos ?? [];
  if (photos.length === 0) {
    return (
      <div
        className="flex flex-col items-center gap-2 rounded-lg py-14"
        style={{ backgroundColor: "#121814", border: "1px dashed #1F2A24" }}
      >
        <ImageIcon size={24} style={{ color: "#4A5A50" }} />
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          The album is empty.
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
      {photos.map((p) => (
        <AlbumPhotoCard key={p.id} photo={publicSafePhoto(p)} />
      ))}
    </div>
  );
}

// Scrubs fields the public viewer must never surface. If AlbumPhotoCard
// grows to display more, add a guard here first.
function publicSafePhoto(p: AlbumPhoto): AlbumPhoto {
  return {
    ...p,
    added_by: "",
    latitude: undefined,
    longitude: undefined,
  };
}

export function PublicAlbumError({ kind }: { kind: PublicAlbumErrorKind }) {
  const msg = errorCopy(kind);
  return (
    <div
      className="flex flex-col items-center gap-3 rounded-2xl py-14 px-6 text-center"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Link2Off size={26} style={{ color: "#8B9A8E" }} />
      <div className="flex flex-col gap-1">
        <h1
          className="text-lg tracking-tight"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "#ECEFEA",
          }}
        >
          {msg.title}
        </h1>
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {msg.body}
        </p>
      </div>
    </div>
  );
}

function errorCopy(kind: PublicAlbumErrorKind): {
  title: string;
  body: string;
} {
  switch (kind) {
    case "invalid":
      return {
        title: "Album not found",
        body: "This link isn't valid. Double-check the URL, or ask the sharer for a new link.",
      };
    case "gone":
      return {
        title: "This album is no longer available",
        body: "The share link has been revoked or has expired. Ask the sharer for a fresh link.",
      };
    case "error":
      return {
        title: "Something went wrong",
        body: "We couldn't load this album right now. Please try again in a moment.",
      };
  }
}

function Message({
  children,
  tone,
}: {
  children: React.ReactNode;
  tone: "muted";
}) {
  return (
    <p
      className="text-sm text-center py-8"
      style={{ color: tone === "muted" ? "#8B9A8E" : "#ECEFEA" }}
    >
      {children}
    </p>
  );
}
