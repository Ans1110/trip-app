"use client";

import { useEffect, useMemo, useRef } from "react";

import type { ChatMessage } from "@/lib/apis/chat-api";

import { ChatMediaAttachment } from "./chat-media";

export function ChatMessageList({
  messages,
  myId,
  loading,
  error,
}: {
  messages: ChatMessage[];
  myId: string | null | undefined;
  loading: boolean;
  error: string | null;
}) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  // Server returns DESC; render OLDEST at top → newest at bottom for readability.
  const ordered = useMemo(() => [...messages].reverse(), [messages]);
  const newestId = ordered.length > 0 ? ordered[ordered.length - 1].id : null;

  useEffect(() => {
    // Auto-scroll to bottom whenever the newest id changes.
    const el = scrollerRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [newestId]);

  return (
    <div
      ref={scrollerRef}
      className="chat-scroll flex-1 min-h-0 overflow-y-auto p-4 flex flex-col gap-2"
      style={{ backgroundColor: "#0B100D" }}
    >
      {loading && (
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {error && (
        <p className="text-xs" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      )}
      {!loading && !error && ordered.length === 0 && (
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          No messages yet. Say hi.
        </p>
      )}
      {ordered.map((m) => (
        <MessageBubble key={m.id} message={m} mine={m.sender_id === myId} />
      ))}
    </div>
  );
}

function MessageBubble({
  message,
  mine,
}: {
  message: ChatMessage;
  mine: boolean;
}) {
  const time = new Date(message.created_at).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
  return (
    <div
      className="flex flex-col max-w-[80%] gap-1"
      style={{ alignSelf: mine ? "flex-end" : "flex-start" }}
    >
      <div
        className="px-3 py-2 rounded-2xl text-sm whitespace-pre-wrap break-words flex flex-col gap-2"
        style={{
          backgroundColor: mine
            ? "color-mix(in srgb, var(--season-button) 22%, transparent)"
            : "#121814",
          color: "#ECEFEA",
          border: mine
            ? "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)"
            : "1px solid #1F2A24",
        }}
      >
        {message.media_id && (
          <ChatMediaAttachment
            mediaId={message.media_id}
            mediaMime={message.media_mime}
            mediaBytes={message.media_bytes}
            mediaFilename={message.media_filename}
          />
        )}
        {message.content && <span>{message.content}</span>}
      </div>
      <span
        className="text-[10px]"
        style={{
          color: "#8B9A8E",
          alignSelf: mine ? "flex-end" : "flex-start",
        }}
      >
        {time}
      </span>
    </div>
  );
}
