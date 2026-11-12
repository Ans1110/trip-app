"use client";

import { useRef } from "react";
import { AlertCircle, CheckCircle2, Loader2, RotateCw, Upload, X } from "lucide-react";

import type { UploadItemState, UseAlbumUploadResult } from "@/hooks/album-hooks";
import type { UploadPhase } from "@/lib/album/upload";
import { cn } from "@/lib/utils";

// Accepts HEIC/HEIF too; heic-convert.ts detects and converts before upload.
// Backend rejects HEIC uploads, so conversion is mandatory on the client.
const FILE_INPUT_ACCEPT =
  "image/jpeg,image/png,image/webp,image/gif,image/heic,image/heif,.heic,.heif";

export function AlbumUploader({ upload }: { upload: UseAlbumUploadResult }) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  const onFiles = (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList);
    void upload.upload(files);
    if (inputRef.current) inputRef.current.value = "";
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        {/* A label opens the picker natively; scripted input.click() on a display:none input can be refused by the browser. */}
        <label
          aria-disabled={upload.isRunning || undefined}
          className={cn(
            "inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium cursor-pointer select-none has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring",
            upload.isRunning && "opacity-50 pointer-events-none",
          )}
          style={{
            backgroundColor: "color-mix(in srgb, var(--mint) 22%, transparent)",
            color: "var(--foreground)",
            border: "1px solid var(--border)",
          }}
        >
          <input
            ref={inputRef}
            type="file"
            multiple
            accept={FILE_INPUT_ACCEPT}
            disabled={upload.isRunning}
            onChange={(e) => onFiles(e.currentTarget.files)}
            className="sr-only"
          />
          <Upload size={14} aria-hidden /> Add photos
        </label>
        {upload.items.length > 0 && (
          <button
            type="button"
            onClick={upload.clearAll}
            className="text-xs"
            style={{ color: "var(--muted-foreground)" }}
          >
            Clear finished
          </button>
        )}
      </div>

      {upload.items.length > 0 && (
        <ul className="flex flex-col gap-2">
          {upload.items.map((item) => (
            <UploadRow
              key={item.ticket.id}
              item={item}
              onRetry={() => void upload.retry(item.ticket.id)}
              onDismiss={() => upload.clear(item.ticket.id)}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function UploadRow({
  item,
  onRetry,
  onDismiss,
}: {
  item: UploadItemState;
  onRetry: () => void;
  onDismiss: () => void;
}) {
  const pct = item.total > 0 ? Math.min(100, (item.loaded / item.total) * 100) : 0;

  return (
    <li
      className="flex items-center gap-3 rounded-lg px-3 py-2"
      style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
    >
      <PhaseIcon phase={item.phase} />
      <div className="flex flex-col flex-1 min-w-0">
        <span className="text-xs truncate" style={{ color: "var(--foreground)" }}>
          {item.ticket.originalFile.name}
        </span>
        <div className="mt-1 flex items-center gap-2">
          <span className="text-[10px]" style={{ color: "var(--muted-foreground)" }}>
            {phaseLabel(item.phase)}
          </span>
          {item.phase === "uploading" && (
            <div
              className="h-1 flex-1 rounded-full overflow-hidden"
              style={{ backgroundColor: "var(--muted)" }}
            >
              <div
                className="h-full transition-all"
                style={{
                  width: `${pct}%`,
                  backgroundColor: "color-mix(in srgb, var(--season-button) 60%, transparent)",
                }}
              />
            </div>
          )}
          {item.error && (
            <span className="text-[10px]" style={{ color: "var(--destructive)" }}>
              {item.error.message}
            </span>
          )}
        </div>
      </div>
      {item.phase === "error" && (
        <button
          type="button"
          onClick={onRetry}
          title="Retry"
          className="text-xs inline-flex items-center gap-1"
          style={{ color: "var(--foreground)" }}
        >
          <RotateCw size={12} /> Retry
        </button>
      )}
      {(item.phase === "done" || item.phase === "error") && (
        <button
          type="button"
          onClick={onDismiss}
          title="Dismiss"
          style={{ color: "var(--muted-foreground)" }}
        >
          <X size={14} />
        </button>
      )}
    </li>
  );
}

function PhaseIcon({ phase }: { phase: UploadPhase }) {
  if (phase === "done") {
    return <CheckCircle2 size={14} style={{ color: "var(--mint-ink)" }} />;
  }
  if (phase === "error") {
    return <AlertCircle size={14} style={{ color: "var(--destructive)" }} />;
  }
  return <Loader2 size={14} className="animate-spin" style={{ color: "var(--muted-foreground)" }} />;
}

function phaseLabel(phase: UploadPhase): string {
  switch (phase) {
    case "queued":
      return "Queued";
    case "converting":
      return "Converting HEIC";
    case "initiating":
      return "Preparing";
    case "uploading":
      return "Uploading";
    case "completing":
      return "Finalising";
    case "done":
      return "Done";
    case "error":
      return "Failed";
  }
}
