"use client";

import { useMemo, useState } from "react";
import { ImageIcon, Share2 } from "lucide-react";

import {
  errorMessage,
  useAlbumPhotos,
  useAlbumUpload,
} from "@/hooks/album-hooks";
import { useRoom } from "@/hooks/trip-hooks";
import type { AlbumPhoto } from "@/lib/apis/album-api";
import type { UserSummary } from "@/lib/apis/friend-api";

import { AlbumCaptionModal } from "./album-caption-modal";
import { AlbumDeleteConfirmModal } from "./album-delete-confirm-modal";
import { AlbumPhotoCard } from "./album-photo-card";
import { AlbumShareManagerModal } from "./album-share-manager";
import { AlbumUploader } from "./album-uploader";

type ActiveModal =
  | { kind: "none" }
  | { kind: "caption"; photo: AlbumPhoto }
  | { kind: "delete"; photo: AlbumPhoto }
  | { kind: "share" };

export function AlbumPanel({ tripId }: { tripId: string }) {
  const photos = useAlbumPhotos(tripId);
  const upload = useAlbumUpload(tripId);
  const room = useRoom(tripId);
  const [modal, setModal] = useState<ActiveModal>({ kind: "none" });

  const memberById = useMemo(() => {
    const map = new Map<string, UserSummary>();
    for (const m of room.data?.members ?? []) {
      map.set(m.user.id, m.user);
    }
    return map;
  }, [room.data]);

  const close = () => setModal({ kind: "none" });

  return (
    <section className="flex flex-col gap-5">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <AlbumUploader upload={upload} />
        <button
          type="button"
          onClick={() => setModal({ kind: "share" })}
          className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium"
          style={{
            backgroundColor:
              "color-mix(in srgb, var(--season-button) 22%, transparent)",
            color: "#ECEFEA",
            border: "1px solid #1F2A24",
          }}
        >
          <Share2 size={14} /> Share album
        </button>
      </div>

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
            <AlbumPhotoCard
              key={p.id}
              photo={p}
              uploader={memberById.get(p.added_by)}
              onEdit={(photo) => setModal({ kind: "caption", photo })}
              onDelete={(photo) => setModal({ kind: "delete", photo })}
            />
          ))}
        </div>
      )}

      {modal.kind === "caption" && (
        <AlbumCaptionModal photo={modal.photo} onClose={close} />
      )}
      {modal.kind === "delete" && (
        <AlbumDeleteConfirmModal photo={modal.photo} onClose={close} />
      )}
      {modal.kind === "share" && (
        <AlbumShareManagerModal tripId={tripId} onClose={close} />
      )}
    </section>
  );
}
