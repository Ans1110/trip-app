"use client";

import { useRef } from "react";
import { AlertCircle, CheckCircle2, Loader2, RotateCw, Upload, X } from "lucide-react";

import type { UploadItemState, UseAlbumUploadResult } from "@/hooks/album-hooks";
import type { UploadPhase } from "@/lib/album/upload";

// Accepts HEIC/HEIF too; heic-convert.ts detects and converts before upload.
// Backend rejects HEIC uploads, so conversion is mandatory on the client.
const FILE_INPUT_ACCEPT =
  "image/jpeg,image/png,image/webp,image/gif,image/heic,image/heif,.heic,.heif";

export function AlbumUploader({ upload }: { upload: UseAlbumUploadResult }) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  const onPick = () => inputRef.current?.click();

  const onFiles = (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList);
    void upload.upload(files);
    if (inputRef.current) inputRef.current.value = "";
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onPick}
          disabled={upload.isRunning}
          className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium disabled:opacity-50"
          style={{
            backgroundColor: "color-mix(in srgb, var(--season-button) 22%, transparent)",
            color: "#ECEFEA",
            border: "1px solid #1F2A24",
          }}
        >
          <Upload size={14} /> Add photos
        </button>
        {upload.items.length > 0 && (
          <button
            type="button"
            onClick={upload.clearAll}
            className="text-xs"
            style={{ color: "#8B9A8E" }}
          >
            Clear finished
          </button>
        )}
      </div>

      <input
        ref={inputRef}
        type="file"
        multiple
        accept={FILE_INPUT_ACCEPT}
        onChange={(e) => onFiles(e.currentTarget.files)}
        className="hidden"
      />

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
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <PhaseIcon phase={item.phase} />
      <div className="flex flex-col flex-1 min-w-0">
        <span className="text-xs truncate" style={{ color: "#ECEFEA" }}>
          {item.ticket.originalFile.name}
        </span>
        <div className="mt-1 flex items-center gap-2">
          <span className="text-[10px]" style={{ color: "#8B9A8E" }}>
            {phaseLabel(item.phase)}
          </span>
          {item.phase === "uploading" && (
            <div
              className="h-1 flex-1 rounded-full overflow-hidden"
              style={{ backgroundColor: "#1F2A24" }}
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
            <span className="text-[10px]" style={{ color: "#FCA5A5" }}>
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
          style={{ color: "#ECEFEA" }}
        >
          <RotateCw size={12} /> Retry
        </button>
      )}
      {(item.phase === "done" || item.phase === "error") && (
        <button
          type="button"
          onClick={onDismiss}
          title="Dismiss"
          style={{ color: "#8B9A8E" }}
        >
          <X size={14} />
        </button>
      )}
    </li>
  );
}

function PhaseIcon({ phase }: { phase: UploadPhase }) {
  if (phase === "done") {
    return <CheckCircle2 size={14} style={{ color: "#86EFAC" }} />;
  }
  if (phase === "error") {
    return <AlertCircle size={14} style={{ color: "#FCA5A5" }} />;
  }
  return <Loader2 size={14} className="animate-spin" style={{ color: "#8B9A8E" }} />;
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
