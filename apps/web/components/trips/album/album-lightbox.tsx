"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Download, Loader2, Maximize2, Minimize2, X } from "lucide-react";

import { useSaveAlbumPhoto } from "@/hooks/album-hooks";
import { albumApi, type AlbumPhoto } from "@/lib/apis/album-api";
import { cn } from "@/lib/utils";

type Props = {
  photos: AlbumPhoto[];
  index: number;
  onIndexChange: (i: number) => void;
  onClose: () => void;
};

export function AlbumLightbox({ photos, index, onIndexChange, onClose }: Props) {
  const photo = photos[index];
  const count = photos.length;
  const [actualSize, setActualSize] = useState(false);
  const closeRef = useRef<HTMLButtonElement | null>(null);
  const touchX = useRef<number | null>(null);
  const save = useSaveAlbumPhoto();

  const go = useCallback(
    (delta: number) => {
      if (count < 2) return;
      setActualSize(false);
      onIndexChange((index + delta + count) % count);
    },
    [count, index, onIndexChange],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      else if (e.key === "ArrowRight") go(1);
      else if (e.key === "ArrowLeft") go(-1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go, onClose]);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const { overflow } = document.body.style;
    document.body.style.overflow = "hidden";
    closeRef.current?.focus();
    return () => {
      document.body.style.overflow = overflow;
      previous?.focus?.();
    };
  }, []);

  if (!photo) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={`Photo ${index + 1} of ${count}`}
      className="album-lightbox fixed inset-0 z-50 flex flex-col"
      onTouchStart={(e) => {
        touchX.current = actualSize ? null : e.touches[0].clientX;
      }}
      onTouchEnd={(e) => {
        if (touchX.current === null) return;
        const dx = e.changedTouches[0].clientX - touchX.current;
        touchX.current = null;
        if (Math.abs(dx) > 50) go(dx < 0 ? 1 : -1);
      }}
    >
      <header className="relative z-10 flex items-center gap-2 px-3 sm:px-5 h-14 shrink-0 text-white">
        <span className="text-sm tabular-nums text-white/80">
          {index + 1} / {count}
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <ToolbarButton
            label={actualSize ? "Fit to screen" : "View actual size"}
            onClick={() => setActualSize((v) => !v)}
          >
            {actualSize ? <Minimize2 className="size-[18px]" /> : <Maximize2 className="size-[18px]" />}
          </ToolbarButton>
          <ToolbarButton
            label={save.isError ? "Couldn't save, try again" : "Save photo"}
            onClick={() => save.mutate(photo)}
            busy={save.isPending}
          >
            {save.isPending ? <Loader2 className="size-[18px] animate-spin" /> : <Download className="size-[18px]" />}
          </ToolbarButton>
          <ToolbarButton label="Close" onClick={onClose} ref={closeRef}>
            <X className="size-5" />
          </ToolbarButton>
        </div>
      </header>

      <div
        className={cn(
          "relative flex-1 min-h-0",
          actualSize ? "overflow-auto overscroll-contain" : "flex items-center justify-center px-3 pb-6 sm:px-16",
        )}
        onClick={(e) => {
          if (e.target === e.currentTarget) onClose();
        }}
      >
        <LightboxImage
          key={photo.id}
          photo={photo}
          actualSize={actualSize}
          onToggle={() => setActualSize((v) => !v)}
        />
      </div>

      {count > 1 && !actualSize && (
        <>
          <NavButton side="left" label="Previous photo" onClick={() => go(-1)} />
          <NavButton side="right" label="Next photo" onClick={() => go(1)} />
        </>
      )}
    </div>
  );
}

function LightboxImage({
  photo,
  actualSize,
  onToggle,
}: {
  photo: AlbumPhoto;
  actualSize: boolean;
  onToggle: () => void;
}) {
  const [src, setSrc] = useState(photo.original_url);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const retried = useRef(false);
  const preview = photo.thumb_medium_url || photo.thumb_small_url;

  // Presigned URLs from the list can expire while the album stays open.
  const onError = () => {
    if (retried.current) {
      setFailed(true);
      return;
    }
    retried.current = true;
    albumApi
      .downloadOriginal(photo.id)
      .then((r) => setSrc(r.url))
      .catch(() => setFailed(true));
  };

  if (failed) {
    return <p className="m-auto text-sm text-white/80">This photo couldn&rsquo;t be loaded.</p>;
  }

  return (
    <div className={cn("relative", actualSize ? "w-max min-w-full min-h-full grid place-items-center" : "max-h-full max-w-full")}>
      {!loaded && preview && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={preview}
          alt=""
          aria-hidden
          className="absolute inset-0 m-auto max-h-full max-w-full object-contain blur-sm"
        />
      )}
      {!loaded && (
        <Loader2 className="absolute inset-0 m-auto size-7 animate-spin text-white/80" aria-label="Loading photo" />
      )}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={src}
        alt="Album photo"
        decoding="async"
        onLoad={() => setLoaded(true)}
        onError={onError}
        onClick={onToggle}
        className={cn(
          "relative block select-none transition-opacity duration-200",
          loaded ? "opacity-100" : "opacity-0",
          actualSize
            ? "max-w-none cursor-zoom-out"
            : "max-h-[calc(100dvh-5.5rem)] max-w-full object-contain cursor-zoom-in rounded-md",
        )}
        draggable={false}
      />
    </div>
  );
}

function ToolbarButton({
  label,
  onClick,
  busy,
  children,
  ref,
}: {
  label: string;
  onClick: () => void;
  busy?: boolean;
  children: React.ReactNode;
  ref?: React.Ref<HTMLButtonElement>;
}) {
  return (
    <button
      ref={ref}
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      disabled={busy}
      aria-busy={busy || undefined}
      className="grid place-items-center size-10 rounded-full text-white hover:bg-white/15 outline-none focus-visible:ring-2 focus-visible:ring-white/80 disabled:cursor-wait"
    >
      {children}
    </button>
  );
}

function NavButton({ side, label, onClick }: { side: "left" | "right"; label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className={cn(
        "absolute top-1/2 -translate-y-1/2 z-10 hidden sm:grid place-items-center size-11 rounded-full bg-white/12 text-white backdrop-blur hover:bg-white/25 outline-none focus-visible:ring-2 focus-visible:ring-white/80",
        side === "left" ? "left-3" : "right-3",
      )}
    >
      {side === "left" ? <ChevronLeft className="size-6" /> : <ChevronRight className="size-6" />}
    </button>
  );
}
