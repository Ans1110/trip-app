"use client";

import { MessageCircle } from "lucide-react";
import { useEffect, useMemo } from "react";

import { useMe } from "@/hooks/auth-hooks";
import { useChatRooms } from "@/hooks/chat-hooks";
import { useChatWindowStore } from "@/store/chat-window-store";

import { ChatWindow } from "./chat-window";

export function ChatFab() {
  const meQuery = useMe();
  const signedIn = meQuery.isFetched && !!meQuery.data;

  const open = useChatWindowStore((s) => s.open);
  const openWindow = useChatWindowStore((s) => s.openWindow);
  const close = useChatWindowStore((s) => s.close);

  const roomsQuery = useChatRooms({
    enabled: signedIn,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  });

  const unread = useMemo(
    () =>
      (roomsQuery.data ?? []).reduce(
        (sum, r) => sum + (r.unread_count ?? 0),
        0,
      ),
    [roomsQuery.data],
  );
  const hasUnread = unread > 0;
  const badgeLabel = unread > 9 ? "9+" : String(unread);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, close]);

  if (!signedIn) return null;

  const ariaLabel = hasUnread
    ? `Open chat, ${unread} unread`
    : "Open chat";

  return (
    <>
      <button
        type="button"
        onClick={openWindow}
        aria-label={ariaLabel}
        // Sits to the left of the FriendsFab (which is at right-6).
        className="season-transition fixed bottom-6 right-24 z-40 inline-flex items-center justify-center size-14 rounded-full hover:-translate-y-0.5 active:translate-y-0"
        style={{
          backgroundColor: "var(--season-button)",
          color: "#0B100D",
          boxShadow: "0 12px 32px var(--season-button-shadow)",
        }}
      >
        <MessageCircle className="size-6" />
        {hasUnread && (
          <span
            aria-hidden
            className="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full text-[11px] font-semibold leading-none"
            style={{
              backgroundColor: "#E5484D",
              color: "#FFFFFF",
              border: "2px solid #0B100D",
            }}
          >
            {badgeLabel}
          </span>
        )}
      </button>

      {open && <ChatWindow />}
    </>
  );
}
