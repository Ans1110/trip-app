"use client";

import { ArrowLeft, X } from "lucide-react";
import { useEffect, useMemo } from "react";

import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useChatMessages,
  useChatRealtime,
  useChatRooms,
  useMarkRead,
} from "@/hooks/chat-hooks";
import type { ChatRoom as ChatRoomType } from "@/lib/apis/chat-api";

import { ChatComposer } from "./chat-composer";
import { ChatMessageList } from "./chat-message-list";

function roomTitle(room: ChatRoomType | undefined): string {
  if (!room) return "Chat";
  if (room.type === "dm") {
    return room.peer?.name || room.peer?.email || "Direct message";
  }
  return room.name || "Group chat";
}

const CONNECTION_LABEL: Record<string, string> = {
  idle: "Idle",
  connecting: "Connecting…",
  open: "Connected",
  reconnecting: "Reconnecting…",
  closed: "Disconnected",
};

export function ChatRoom({
  roomId,
  onBack,
  onClose,
}: {
  roomId: string;
  onBack?: () => void;
  onClose?: () => void;
}) {
  const meQuery = useMe();
  const roomsQuery = useChatRooms();
  const messagesQuery = useChatMessages(roomId, { limit: 50 });
  const markRead = useMarkRead();

  const room = useMemo(
    () => roomsQuery.data?.find((r) => r.id === roomId),
    [roomsQuery.data, roomId],
  );

  const { state, send } = useChatRealtime(roomId);

  // Auto-advance read pointer to the newest visible message. Server returns
  // messages DESC so index 0 is the newest.
  const newest = messagesQuery.data?.[0]?.id;
  useEffect(() => {
    if (!newest) return;
    markRead.mutate({ roomId, input: { last_read_id: newest } });
    // Intentionally exclude markRead: it changes identity every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [newest, roomId]);

  const canSend = state === "open";

  return (
    <div className="flex flex-col h-full min-h-0 overflow-hidden">
      <header
        className="flex items-center gap-2 px-3 py-3"
        style={{
          backgroundColor: "#121814",
          borderBottom: "1px solid #1F2A24",
        }}
      >
        {onBack && (
          <button
            type="button"
            onClick={onBack}
            aria-label="Back to chat list"
            className="size-7 inline-flex items-center justify-center rounded-full shrink-0"
            style={{ color: "#8B9A8E" }}
          >
            <ArrowLeft className="size-4" />
          </button>
        )}
        <div className="flex-1 min-w-0 flex flex-col">
          <h2
            className="text-sm font-medium truncate"
            style={{ color: "#ECEFEA" }}
          >
            {roomTitle(room)}
          </h2>
          {room?.type === "dm" && (
            <span className="text-[10px]" style={{ color: "#8B9A8E" }}>
              Direct message
            </span>
          )}
        </div>
        <span
          className="text-[10px] px-2 py-0.5 rounded-full shrink-0"
          style={{
            color: state === "open" ? "#7BD88F" : "#8B9A8E",
            border: "1px solid #1F2A24",
          }}
        >
          {CONNECTION_LABEL[state] ?? state}
        </span>
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            aria-label="Close chat"
            className="size-7 inline-flex items-center justify-center rounded-full shrink-0"
            style={{ color: "#8B9A8E" }}
          >
            <X className="size-4" />
          </button>
        )}
      </header>

      <ChatMessageList
        messages={messagesQuery.data ?? []}
        myId={meQuery.data?.id}
        loading={messagesQuery.isLoading}
        error={errorMessage(messagesQuery.error)}
      />

      <div className="p-3" style={{ backgroundColor: "#0B100D" }}>
        <ChatComposer roomId={roomId} disabled={!canSend} onSend={send} />
      </div>
    </div>
  );
}
