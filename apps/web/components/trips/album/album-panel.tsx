"use client";

import { ImageIcon } from "lucide-react";

import {
  errorMessage,
  useAlbumPhotos,
  useAlbumUpload,
} from "@/hooks/album-hooks";

import { AlbumPhotoCard } from "./album-photo-card";
import { AlbumUploader } from "./album-uploader";

export function AlbumPanel({ tripId }: { tripId: string }) {
  const photos = useAlbumPhotos(tripId);
  const upload = useAlbumUpload(tripId);

  return (
    <section className="flex flex-col gap-5">
      <AlbumUploader upload={upload} />

      {photos.isLoading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading photos…
        </p>
      )}

      {photos.isError && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(photos.error) ?? "Failed to load photos."}
        </p>
      )}

      {photos.data && photos.data.length === 0 && !upload.isRunning && (
        <div
          className="flex flex-col items-center gap-2 rounded-lg py-10"
          style={{
            backgroundColor: "#121814",
            border: "1px dashed #1F2A24",
          }}
        >
          <ImageIcon size={22} style={{ color: "#4A5A50" }} />
          <p className="text-xs" style={{ color: "#8B9A8E" }}>
            No photos yet — upload from your phone or camera roll.
          </p>
        </div>
      )}

      {photos.data && photos.data.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
          {photos.data.map((p) => (
            <AlbumPhotoCard key={p.id} photo={p} />
          ))}
        </div>
      )}
    </section>
  );
}
