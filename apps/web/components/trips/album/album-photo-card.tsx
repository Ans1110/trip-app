"use client";

import { useState } from "react";
import { Download, ImageOff, Loader2, Trash2 } from "lucide-react";

import { useSaveAlbumPhoto } from "@/hooks/album-hooks";

import type { AlbumPhoto } from "@/lib/apis/album-api";
import type { UserSummary } from "@/lib/apis/friend-api";

export type AlbumPhotoCardProps = {
  photo: AlbumPhoto;
  uploader?: UserSummary;
  onOpen?: () => void;
  onDelete?: (photo: AlbumPhoto) => void;
};

// Uses the small thumbnail so a 30-photo grid doesn't trigger 30 originals;
// falls back to a placeholder when the thumbnail is missing (uploads can race
// the thumbnail worker) or the presigned URL has expired.
export function AlbumPhotoCard({
  photo,
  uploader,
  onOpen,
  onDelete,
}: AlbumPhotoCardProps) {
  const [failed, setFailed] = useState(false);
  const thumb = photo.thumb_small_url;
  const save = useSaveAlbumPhoto();

  if (!thumb || failed) {
    return (
      <div
        className="aspect-square flex flex-col items-center justify-center gap-1 rounded-lg"
        style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
      >
        <ImageOff size={20} style={{ color: "var(--subtle-foreground)" }} />
      </div>
    );
  }

  return (
    <div
      className="group relative aspect-square overflow-hidden rounded-lg"
      style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
    >
      {/* Native <img> because the presigned S3 URL host isn't in next/image's
          domain allowlist and rotates per config. */}
      <button
        type="button"
        onClick={onOpen}
        aria-label="Open photo"
        className="block size-full cursor-zoom-in outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
      >
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={thumb}
          alt="Album photo"
          loading="lazy"
          decoding="async"
          onError={() => setFailed(true)}
          className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
        />
      </button>

      {uploader && (
        <div className="pointer-events-none absolute top-1.5 left-1.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
          <UploaderAvatar user={uploader} />
        </div>
      )}

      <div className="pointer-events-none absolute inset-0 flex items-start justify-end p-1.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100 [@media(hover:none)]:opacity-100">
        <div className="pointer-events-auto flex items-center gap-1">
          <IconButton
            label={save.isError ? "Couldn't save, try again" : "Save photo"}
            onClick={() => save.mutate(photo)}
            busy={save.isPending}
            tone={save.isError ? "danger" : undefined}
            icon={save.isPending ? <Loader2 size={12} className="animate-spin" /> : <Download size={12} />}
          />
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
        backgroundColor: "color-mix(in oklch, var(--card) 88%, transparent)",
        color: "var(--foreground)",
        border: "1px solid var(--border)",
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
  busy,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  tone?: "danger";
  busy?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      aria-busy={busy || undefined}
      aria-label={label}
      title={label}
      className="inline-flex items-center justify-center size-7 rounded-full backdrop-blur outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-wait"
      style={{
        backgroundColor: "color-mix(in oklch, var(--card) 88%, transparent)",
        color: tone === "danger" ? "var(--destructive)" : "var(--foreground)",
        border: "1px solid var(--border)",
      }}
    >
      {icon}
    </button>
  );
}
