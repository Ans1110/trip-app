"use client";

import { Trash2, X } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { chatApi, type ChatRoom } from "@/lib/apis/chat-api";
import {
  chatKeys,
  errorMessage,
  useChatRooms,
  useEnsureDM,
  useHideRoom,
} from "@/hooks/chat-hooks";
import { useTrips } from "@/hooks/trip-hooks";
import { useChatWindowStore } from "@/store/chat-window-store";

import { ChatRoom as ChatRoomView } from "./chat-room";

export function ChatWindow() {
  const selectedRoomId = useChatWindowStore((s) => s.selectedRoomId);
  const selectRoom = useChatWindowStore((s) => s.selectRoom);
  const close = useChatWindowStore((s) => s.close);

  // Tight poll while the window is open so a DM re-surfacing on the other side
  // (via UnhideRooms in writer.flush) appears without a manual refresh. The
  // FAB keeps a slower background poll for the unread badge.
  const roomsQuery = useChatRooms({
    refetchInterval: 5_000,
    refetchOnWindowFocus: true,
  });
  const rooms = roomsQuery.data ?? [];
  const active = rooms.find((r) => r.id === selectedRoomId) ?? null;

  useTripRoomSync(roomsQuery.data);
  usePendingPeerResolver();

  return (
    <aside
      role="dialog"
      aria-label="Chat"
      // Anchored above the FABs. Small on desktop, near-fullscreen on phones.
      className="fixed bottom-24 right-6 z-50 w-[calc(100vw-24px)] max-w-[380px] h-[calc(100vh-160px)] max-h-[520px] flex flex-col rounded-2xl overflow-hidden"
      style={{
        backgroundColor: "#0B100D",
        border: "1px solid #1F2A24",
        boxShadow: "0 24px 64px rgba(0,0,0,0.5)",
      }}
    >
      {active ? (
        <ChatRoomView
          roomId={active.id}
          onBack={() => selectRoom(null)}
          onClose={close}
        />
      ) : (
        <RoomListView
          rooms={rooms}
          loading={roomsQuery.isLoading}
          error={errorMessage(roomsQuery.error)}
          onSelect={(id) => selectRoom(id)}
          onClose={close}
        />
      )}
    </aside>
  );
}

// -------- Room list view --------

function RoomListView({
  rooms,
  loading,
  error,
  onSelect,
  onClose,
}: {
  rooms: ChatRoom[];
  loading: boolean;
  error: string | null;
  onSelect: (id: string) => void;
  onClose: () => void;
}) {
  const sorted = useMemo(() => sortRooms(rooms), [rooms]);
  return (
    <div className="flex flex-col h-full min-h-0">
      <header
        className="flex items-center justify-between px-4 py-3"
        style={{
          backgroundColor: "#121814",
          borderBottom: "1px solid #1F2A24",
        }}
      >
        <h2 className="text-sm font-medium" style={{ color: "#ECEFEA" }}>
          Chats
        </h2>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close chat"
          className="size-7 inline-flex items-center justify-center rounded-full"
          style={{ color: "#8B9A8E" }}
        >
          <X className="size-4" />
        </button>
      </header>

      <div className="chat-scroll flex-1 min-h-0 overflow-y-auto">
        {loading && (
          <p className="p-4 text-xs" style={{ color: "#8B9A8E" }}>
            Loading…
          </p>
        )}
        {error && (
          <p className="p-4 text-xs" style={{ color: "#FCA5A5" }}>
            {error}
          </p>
        )}
        {!loading && !error && sorted.length === 0 && (
          <p className="p-4 text-xs" style={{ color: "#8B9A8E" }}>
            No chats yet. Start one from a trip or a friend.
          </p>
        )}
        <ul className="flex flex-col">
          {sorted.map((room) => (
            <RoomRow key={room.id} room={room} onSelect={() => onSelect(room.id)} />
          ))}
        </ul>
      </div>
    </div>
  );
}

function RoomRow({
  room,
  onSelect,
}: {
  room: ChatRoom;
  onSelect: () => void;
}) {
  const preview = room.last_message?.content ?? "";
  const label =
    room.type === "dm"
      ? room.peer?.name || room.peer?.email || "Direct message"
      : room.name || "Group chat";
  const hideRoom = useHideRoom();
  // Only DMs are hideable today. Group rooms follow trip membership and would
  // just reappear on next refresh — hide the affordance to avoid confusion.
  const canHide = room.type === "dm" && !hideRoom.isPending;

  const onHide = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!canHide) return;
    const ok = window.confirm(
      `Delete chat with ${label}? Messages will be preserved and the chat will reappear if either of you sends a new message.`,
    );
    if (!ok) return;
    hideRoom.mutate(room.id);
  };

  return (
    <li>
      <div
        className="relative w-full flex items-stretch group"
        style={{ borderBottom: "1px solid #1F2A24" }}
      >
        <button
          type="button"
          onClick={onSelect}
          className="flex-1 min-w-0 text-left px-4 py-3 flex flex-col gap-1 hover:bg-[color:color-mix(in_srgb,var(--season-button)_10%,transparent)]"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm truncate" style={{ color: "#ECEFEA" }}>
              {label}
            </span>
            {room.unread_count > 0 && (
              <span
                className="text-[10px] px-1.5 py-0.5 rounded-full shrink-0"
                style={{
                  backgroundColor:
                    "color-mix(in srgb, var(--season-button) 32%, transparent)",
                  color: "#ECEFEA",
                }}
              >
                {room.unread_count}
              </span>
            )}
          </div>
          <span className="text-xs truncate" style={{ color: "#8B9A8E" }}>
            {preview || (room.type === "dm" ? "Say hi" : "New chat")}
          </span>
        </button>
        {room.type === "dm" && (
          <button
            type="button"
            onClick={onHide}
            aria-label="Delete chat"
            title="Delete chat (messages preserved)"
            disabled={!canHide}
            className="shrink-0 px-3 inline-flex items-center justify-center opacity-0 group-hover:opacity-100 focus:opacity-100 disabled:opacity-30 transition-opacity"
            style={{ color: "#8B9A8E" }}
          >
            <Trash2 className="size-4" />
          </button>
        )}
      </div>
    </li>
  );
}

function sortRooms(rooms: ChatRoom[]): ChatRoom[] {
  return [...rooms].sort((a, b) => {
    const ta = a.last_message?.created_at ?? a.created_at;
    const tb = b.last_message?.created_at ?? b.created_at;
    return tb.localeCompare(ta);
  });
}

// -------- Effects --------

// useTripRoomSync fires idempotent ensureTripRoom for every trip that doesn't
// already have a corresponding chat.room. Runs once per trip per session; the
// backend returns the existing room if any.
function useTripRoomSync(rooms: ChatRoom[] | undefined) {
  const qc = useQueryClient();
  const tripsQuery = useTrips();
  const attemptedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    const trips = tripsQuery.data;
    if (!trips || !rooms) return;
    const existing = new Set(
      rooms.map((r) => r.trip_id).filter((v): v is string => !!v),
    );
    const missing = trips.filter(
      (t) => !existing.has(t.id) && !attemptedRef.current.has(t.id),
    );
    if (missing.length === 0) return;

    for (const t of missing) attemptedRef.current.add(t.id);

    void Promise.allSettled(
      missing.map((t) => chatApi.ensureTripRoom(t.id)),
    ).then(() => {
      qc.invalidateQueries({ queryKey: chatKeys.rooms });
    });
  }, [tripsQuery.data, rooms, qc]);
}

// usePendingPeerResolver: when the store carries a pendingPeerId (e.g. set by
// the friend-menu "Chat" action), resolve to a DM room and select it.
function usePendingPeerResolver() {
  const pendingPeerId = useChatWindowStore((s) => s.pendingPeerId);
  const clearPendingPeer = useChatWindowStore((s) => s.clearPendingPeer);
  const selectRoom = useChatWindowStore((s) => s.selectRoom);
  const ensureDM = useEnsureDM();

  useEffect(() => {
    if (!pendingPeerId) return;
    // ensureDM is idempotent — server returns existing DM if any.
    ensureDM.mutate(
      { peer_id: pendingPeerId },
      {
        onSuccess: (room) => {
          selectRoom(room.id);
          clearPendingPeer();
        },
        onError: () => {
          clearPendingPeer();
        },
      },
    );
    // ensureDM identity is stable enough here; we only want to fire on peer change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pendingPeerId]);
}
