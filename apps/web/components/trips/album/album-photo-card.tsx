"use client";

import { useState } from "react";
import { ImageOff } from "lucide-react";

import type { AlbumPhoto } from "@/lib/apis/album-api";

// AlbumPhotoCard renders one photo in the grid. Uses the small thumbnail URL
// (400px longest side) so a 30-photo grid never triggers 30 full-resolution
// downloads. Falls back to a placeholder if the thumbnail is missing (some
// uploads race the thumbnail worker) or the presigned URL has expired.
export function AlbumPhotoCard({ photo }: { photo: AlbumPhoto }) {
  const [failed, setFailed] = useState(false);
  const thumb = photo.thumb_small_url;

  if (!thumb || failed) {
    return (
      <div
        className="aspect-square flex items-center justify-center rounded-lg"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <ImageOff size={20} style={{ color: "#4A5A50" }} />
      </div>
    );
  }

  return (
    <div
      className="relative aspect-square overflow-hidden rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      {/* Native <img> because the presigned S3 URL host isn't in next/image's
          domain allowlist and rotates per config. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={thumb}
        alt={photo.caption || "Album photo"}
        loading="lazy"
        decoding="async"
        onError={() => setFailed(true)}
        className="h-full w-full object-cover"
      />
    </div>
  );
}
