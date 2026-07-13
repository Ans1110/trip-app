"use client";

import { Download, FileIcon, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

import { mediaApi } from "@/lib/apis/media-api";

// ChatMediaAttachment resolves a presigned GET URL for the message's media_id
// and renders it inline (image/video preview) or as a download chip (files).
// Presigned URLs are short-lived; we don't cache heavily — each mount fetches
// once, which is fine for the current window's lifetime.
export function ChatMediaAttachment({
  mediaId,
  mediaMime,
  mediaBytes,
  mediaFilename,
}: {
  mediaId: string;
  mediaMime?: string;
  mediaBytes?: number;
  mediaFilename?: string;
}) {
  const [url, setUrl] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const ctrl = new AbortController();
    mediaApi
      .presignRead(mediaId, ctrl.signal)
      .then((res) => {
        if (!cancelled) setUrl(res.url);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "load failed");
      });
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [mediaId]);

  if (error) {
    return (
      <div className="text-xs" style={{ color: "#FCA5A5" }}>
        Failed to load attachment
      </div>
    );
  }
  if (!url) {
    return (
      <div
        className="inline-flex items-center gap-2 text-xs"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-3.5 animate-spin" />
        Loading…
      </div>
    );
  }

  const mime = mediaMime ?? "";
  if (mime.startsWith("image/")) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={url}
        alt="attachment"
        className="rounded-lg max-w-full h-auto"
        style={{ maxHeight: 280 }}
      />
    );
  }
  if (mime.startsWith("video/")) {
    return (
      <video
        src={url}
        controls
        preload="metadata"
        className="rounded-lg max-w-full"
        style={{ maxHeight: 280 }}
      />
    );
  }
  const label = mediaFilename || mime || "file";
  return (
    <a
      href={url}
      // download hint so browsers save with the original name instead of the
      // opaque object key.
      download={mediaFilename || undefined}
      target="_blank"
      rel="noreferrer noopener"
      className="inline-flex items-center gap-2 px-3 py-2 rounded-lg text-xs"
      style={{
        backgroundColor: "#0B100D",
        border: "1px solid #1F2A24",
        color: "#ECEFEA",
      }}
    >
      <FileIcon className="size-3.5" style={{ color: "#8B9A8E" }} />
      <span className="truncate max-w-[180px]" title={label}>
        {label}
      </span>
      {typeof mediaBytes === "number" && (
        <span style={{ color: "#8B9A8E" }}>{formatBytes(mediaBytes)}</span>
      )}
      <Download className="size-3.5" style={{ color: "#8B9A8E" }} />
    </a>
  );
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}
