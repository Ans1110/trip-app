"use client";

import { Loader2, Paperclip, Send, X } from "lucide-react";
import { useCallback, useRef, useState } from "react";

import { mediaApi, type Asset } from "@/lib/apis/media-api";
import type { ChatClientMsg } from "@/lib/realtime/chat-protocol";

const genClientMsgId = (): string => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

// Mirror of the server's chat purpose allow-list. Kept narrow enough to keep
// the picker sensible on mobile; the server enforces the real rule.
const ACCEPT =
  "image/jpeg,image/png,image/webp,image/gif,video/mp4,video/webm,video/quicktime,application/pdf,text/plain,application/zip";

type Attachment = {
  asset: Asset;
  filename: string;
};

export function ChatComposer({
  roomId,
  disabled,
  onSend,
}: {
  roomId: string;
  disabled: boolean;
  onSend: (msg: ChatClientMsg) => boolean;
}) {
  const [value, setValue] = useState("");
  const [attachment, setAttachment] = useState<Attachment | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const pickFile = () => fileRef.current?.click();

  const onFile = useCallback(async (file: File) => {
    setUploadError(null);
    setUploading(true);
    try {
      const asset = await mediaApi.uploadFile("chat", file);
      setAttachment({ asset, filename: file.name });
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : "upload failed");
    } finally {
      setUploading(false);
    }
  }, []);

  const clearAttachment = () => {
    setAttachment(null);
    setUploadError(null);
    if (fileRef.current) fileRef.current.value = "";
  };

  const submit = useCallback(() => {
    const content = value.trim();
    // Allow send when there's either text or a media attachment.
    if (!content && !attachment) return;
    if (uploading) return;
    const clientMsgId = genClientMsgId();
    const ok = onSend({
      type: "SEND_MESSAGE",
      msg_id: clientMsgId,
      room_id: roomId,
      data: {
        content,
        client_msg_id: clientMsgId,
        media_id: attachment?.asset.id,
        media_filename: attachment?.filename,
      },
    });
    if (ok) {
      setValue("");
      clearAttachment();
    }
  }, [attachment, onSend, roomId, uploading, value]);

  const canSend =
    !disabled && !uploading && (value.trim().length > 0 || !!attachment);

  return (
    <form
      className="flex flex-col gap-2 p-3 rounded-2xl"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
      {(attachment || uploading || uploadError) && (
        <div
          className="flex items-center gap-2 px-2 py-1.5 text-xs rounded-lg"
          style={{
            backgroundColor: "#0B100D",
            border: "1px solid #1F2A24",
            color: "#ECEFEA",
          }}
        >
          {uploading && (
            <>
              <Loader2 className="size-3.5 animate-spin" />
              <span style={{ color: "#8B9A8E" }}>Uploading…</span>
            </>
          )}
          {!uploading && attachment && (
            <>
              <Paperclip className="size-3.5" style={{ color: "#8B9A8E" }} />
              <span className="truncate flex-1">{attachment.filename}</span>
              <span style={{ color: "#8B9A8E" }}>
                {formatBytes(attachment.asset.bytes)}
              </span>
              <button
                type="button"
                onClick={clearAttachment}
                aria-label="Remove attachment"
                className="shrink-0"
                style={{ color: "#8B9A8E" }}
              >
                <X className="size-3.5" />
              </button>
            </>
          )}
          {!uploading && !attachment && uploadError && (
            <span style={{ color: "#FCA5A5" }}>{uploadError}</span>
          )}
        </div>
      )}

      <div className="flex items-end gap-2">
        <input
          ref={fileRef}
          type="file"
          accept={ACCEPT}
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void onFile(f);
            // Reset so re-picking the same file still fires change.
            e.target.value = "";
          }}
        />
        <button
          type="button"
          onClick={pickFile}
          disabled={disabled || uploading}
          aria-label="Attach photo, video, or file"
          title="Attach photo, video, or file"
          className="size-8 inline-flex items-center justify-center rounded-full disabled:opacity-40"
          style={{ color: "#8B9A8E" }}
        >
          <Paperclip className="size-4" />
        </button>
        <textarea
          rows={1}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          placeholder={disabled ? "Connecting…" : "Type a message…"}
          disabled={disabled}
          className="flex-1 resize-none bg-transparent outline-none text-sm"
          style={{ color: "#ECEFEA", maxHeight: 120 }}
        />
        <button
          type="submit"
          disabled={!canSend}
          className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full disabled:opacity-50"
          style={{
            backgroundColor:
              "color-mix(in srgb, var(--season-button) 22%, transparent)",
            color: "#ECEFEA",
            border:
              "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)",
          }}
        >
          <Send className="size-3.5" />
          Send
        </button>
      </div>
    </form>
  );
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}
