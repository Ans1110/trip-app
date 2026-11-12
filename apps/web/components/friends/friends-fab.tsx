"use client";

import { useEffect, useState } from "react";
import { Users } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import { useIncomingRequests } from "@/hooks/friend-hooks";

import { FriendsTabs } from "./friends-tabs";

export function FriendsFab() {
  const [open, setOpen] = useState(false);
  const meQuery = useMe();
  const signedIn = meQuery.isFetched && !!meQuery.data;
  const incomingQuery = useIncomingRequests({
    enabled: signedIn,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  });
  const incomingCount = incomingQuery.data?.length ?? 0;
  const hasIncoming = incomingCount > 0;
  const badgeLabel = incomingCount > 9 ? "9+" : String(incomingCount);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  if (!signedIn) return null;

  const ariaLabel = hasIncoming
    ? `Open friends, ${incomingCount} pending request${incomingCount === 1 ? "" : "s"}`
    : "Open friends";

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={ariaLabel}
        className="app-fab fixed right-6 z-40 inline-flex items-center justify-center size-14 rounded-full hover:-translate-y-0.5 active:translate-y-0"
        style={{
          backgroundColor: "var(--season-button)",
          color: "var(--primary-foreground)",
          boxShadow: "0 12px 32px var(--season-button-shadow)",
        }}
      >
        <Users className="size-6" />
        {hasIncoming && (
          <span
            aria-hidden
            className="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full text-[11px] font-semibold leading-none"
            style={{
              backgroundColor: "var(--destructive)",
              color: "#FFFFFF",
              border: "2px solid var(--border)",
            }}
          >
            {badgeLabel}
          </span>
        )}
      </button>

      {open && (
        <div className="fixed inset-0 z-50">
          <button
            type="button"
            aria-label="Close friends"
            onClick={() => setOpen(false)}
            className="absolute inset-0 cursor-default"
            style={{ backgroundColor: "var(--overlay)" }}
          />
          <aside
            className="absolute top-0 right-0 h-full w-full sm:w-[440px]"
            style={{
              backgroundColor: "var(--card)",
              borderLeft: "1px solid var(--border)",
              boxShadow: "-24px 0 48px -12px oklch(0.55 0.09 245 / 0.22)",
            }}
            aria-label="Friends panel"
          >
            <FriendsTabs onClose={() => setOpen(false)} />
          </aside>
        </div>
      )}
    </>
  );
}
