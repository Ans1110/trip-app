"use client";

import { useState } from "react";
import { ImageOff, Pencil, Trash2 } from "lucide-react";

import type { AlbumPhoto } from "@/lib/apis/album-api";
import type { UserSummary } from "@/lib/apis/friend-api";

export type AlbumPhotoCardProps = {
  photo: AlbumPhoto;
  uploader?: UserSummary;
  onEdit?: (photo: AlbumPhoto) => void;
  onDelete?: (photo: AlbumPhoto) => void;
};

// Uses the small thumbnail so a 30-photo grid doesn't trigger 30 originals;
// falls back to a placeholder when the thumbnail is missing (uploads can race
// the thumbnail worker) or the presigned URL has expired.
export function AlbumPhotoCard({
  photo,
  uploader,
  onEdit,
  onDelete,
}: AlbumPhotoCardProps) {
  const [failed, setFailed] = useState(false);
  const thumb = photo.thumb_small_url;
  const canManage = !!onEdit || !!onDelete;

  if (!thumb || failed) {
    return (
      <div
        className="aspect-square flex flex-col items-center justify-center gap-1 rounded-lg"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <ImageOff size={20} style={{ color: "#4A5A50" }} />
        {photo.caption && (
          <p className="text-[10px] px-2 truncate" style={{ color: "#8B9A8E" }}>
            {photo.caption}
          </p>
        )}
      </div>
    );
  }

  return (
    <div
      className="group relative aspect-square overflow-hidden rounded-lg"
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

      {photo.caption && (
        <div
          className="absolute inset-x-0 bottom-0 px-2 py-1.5 text-[11px] leading-snug"
          style={{
            background:
              "linear-gradient(to top, rgba(0,0,0,0.72), transparent)",
            color: "#ECEFEA",
          }}
        >
          <span className="line-clamp-2">{photo.caption}</span>
        </div>
      )}

      {uploader && (
        <div className="pointer-events-none absolute top-1.5 left-1.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
          <UploaderAvatar user={uploader} />
        </div>
      )}

      {canManage && (
        <div className="pointer-events-none absolute inset-0 flex items-start justify-end p-1.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
          <div className="pointer-events-auto flex items-center gap-1">
            {onEdit && (
              <IconButton
                label="Edit caption"
                onClick={() => onEdit(photo)}
                icon={<Pencil size={12} />}
              />
            )}
            {onDelete && (
              <IconButton
                label="Delete photo"
                onClick={() => onDelete(photo)}
                icon={<Trash2 size={12} />}
                tone="danger"
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function UploaderAvatar({ user }: { user: UserSummary }) {
  const initial = (user.name?.trim()[0] ?? "?").toUpperCase();
  return (
    <span
      title={`Uploaded by ${user.name}`}
      aria-label={`Uploaded by ${user.name}`}
      className="inline-flex items-center justify-center size-6 rounded-full text-[10px] font-semibold backdrop-blur overflow-hidden"
      style={{
        backgroundColor: "rgba(11,16,13,0.72)",
        color: "#ECEFEA",
        border: "1px solid #1F2A24",
      }}
    >
      {user.avatar_url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={user.avatar_url} alt="" className="size-full object-cover" />
      ) : (
        initial
      )}
    </span>
  );
}

function IconButton({
  icon,
  label,
  onClick,
  tone,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  tone?: "danger";
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className="inline-flex items-center justify-center size-6 rounded-full backdrop-blur"
      style={{
        backgroundColor: "rgba(11,16,13,0.72)",
        color: tone === "danger" ? "#FCA5A5" : "#ECEFEA",
        border: "1px solid #1F2A24",
      }}
    >
      {icon}
    </button>
  );
}
