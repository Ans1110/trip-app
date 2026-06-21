"use client";

import { useEffect, useRef, useState } from "react";
import { ImagePlus, Loader2, Trash2, Upload, X } from "lucide-react";

import { errorMessage, useUpdateTrip } from "@/hooks/trip-hooks";
import { uploadImage } from "@/lib/cloudinary-upload";
import type { Trip } from "@/lib/trip-api";

const MAX_BYTES = 8 * 1024 * 1024; // 8 MB
const ACCEPTED = ["image/jpeg", "image/png", "image/webp", "image/gif"];

export function CoverImageDrawer({
  trip,
  onClose,
}: {
  trip: Trip;
  onClose: () => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(trip.cover_image ?? null);
  const update = useUpdateTrip();

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !busy) onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, busy]);

  const handlePick = () => inputRef.current?.click();

  const handleFile = async (file: File) => {
    setLocalError(null);
    if (!ACCEPTED.includes(file.type)) {
      setLocalError("Use a JPEG, PNG, WebP, or GIF image.");
      return;
    }
    if (file.size > MAX_BYTES) {
      setLocalError("Image must be 8 MB or smaller.");
      return;
    }
    setBusy(true);
    try {
      const uploaded = await uploadImage(file);
      setPreview(uploaded.secure_url);
      await new Promise<void>((resolve, reject) => {
        update.mutate(
          { id: trip.id, input: { cover_image: uploaded.secure_url } },
          { onSuccess: () => resolve(), onError: (err) => reject(err) },
        );
      });
      onClose();
    } catch (err) {
      setLocalError(
        err instanceof Error ? err.message : "Upload failed. Try again.",
      );
    } finally {
      setBusy(false);
    }
  };

  const handleRemove = () => {
    if (!trip.cover_image) return;
    setLocalError(null);
    setBusy(true);
    update.mutate(
      { id: trip.id, input: { cover_image: "" } },
      {
        onSuccess: () => {
          setBusy(false);
          onClose();
        },
        onError: (err) => {
          setBusy(false);
          setLocalError(errorMessage(err));
        },
      },
    );
  };

  return (
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        aria-label="Close"
        onClick={() => !busy && onClose()}
        className="absolute inset-0 cursor-default"
        style={{ backgroundColor: "rgba(0,0,0,0.55)" }}
      />
      <aside
        role="dialog"
        aria-modal="true"
        aria-labelledby="cover-drawer-title"
        className="absolute right-0 top-0 h-full w-full max-w-md flex flex-col"
        style={{
          backgroundColor: "#0B100D",
          borderLeft: "1px solid #1F2A24",
          boxShadow: "-24px 0 64px rgba(0,0,0,0.5)",
        }}
      >
        <div
          className="flex items-center justify-between px-6 py-4 border-b"
          style={{ borderColor: "#1F2A24" }}
        >
          <h2
            id="cover-drawer-title"
            className="text-lg tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            Trip cover
          </h2>
          <button
            type="button"
            onClick={() => !busy && onClose()}
            disabled={busy}
            aria-label="Close"
            className="inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#8B9A8E" }}
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-5 flex flex-col gap-5">
          <div
            className="rounded-xl overflow-hidden"
            style={{ border: "1px solid #1F2A24", aspectRatio: "16 / 9" }}
          >
            {preview ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={preview}
                alt="Trip cover preview"
                className="w-full h-full object-cover"
              />
            ) : (
              <div
                className="w-full h-full flex flex-col items-center justify-center gap-2"
                style={{
                  background:
                    "linear-gradient(135deg, #1F2A24 0%, #0B100D 100%)",
                  color: "#6B7A6F",
                }}
              >
                <ImagePlus className="size-6" />
                <p className="text-xs">No cover yet</p>
              </div>
            )}
          </div>

          <input
            ref={inputRef}
            type="file"
            accept={ACCEPTED.join(",")}
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) void handleFile(file);
              e.target.value = "";
            }}
          />

          <button
            type="button"
            onClick={handlePick}
            disabled={busy}
            className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
              boxShadow: "0 8px 24px var(--season-button-shadow)",
            }}
          >
            {busy ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Uploading
              </>
            ) : (
              <>
                <Upload className="size-4" />
                {preview ? "Replace image" : "Upload image"}
              </>
            )}
          </button>

          <p className="text-xs" style={{ color: "#6B7A6F" }}>
            JPEG, PNG, WebP, or GIF. Max 8 MB. Uploaded to your Cloudinary
            workspace.
          </p>

          {localError && (
            <p
              className="text-xs rounded-lg px-3 py-2"
              style={{
                color: "#FCA5A5",
                backgroundColor: "rgba(220,38,38,0.08)",
                border: "1px solid rgba(220,38,38,0.25)",
              }}
              role="alert"
            >
              {localError}
            </p>
          )}

          {trip.cover_image && (
            <button
              type="button"
              onClick={handleRemove}
              disabled={busy}
              className="inline-flex items-center justify-center gap-1.5 px-4 py-2 text-xs rounded-full hover:bg-white/5 disabled:opacity-60 self-start"
              style={{ color: "#FCA5A5", border: "1px solid #1F2A24" }}
            >
              <Trash2 className="size-3.5" />
              Remove cover
            </button>
          )}
        </div>
      </aside>
    </div>
  );
}
