"use client";

import { useSyncExternalStore } from "react";
import { LogIn } from "lucide-react";

import { sessionExpiredStore } from "@/lib/session-expired-store";

export function SessionExpiredModal() {
  const open = useSyncExternalStore(
    sessionExpiredStore.subscribe,
    sessionExpiredStore.getSnapshot,
    sessionExpiredStore.getServerSnapshot,
  );

  if (!open) return null;

  const onSignIn = () => {
    sessionExpiredStore.clear();
    const next =
      typeof window !== "undefined"
        ? window.location.pathname + window.location.search
        : "/";
    const url = `/sign-in?next=${encodeURIComponent(next)}`;
    window.location.assign(url);
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="session-expired-title"
      className="fixed inset-0 z-[100] flex items-center justify-center px-4"
      style={{ backgroundColor: "rgba(11,16,13,0.72)" }}
    >
      <div
        className="w-full max-w-sm rounded-2xl p-6 shadow-2xl"
        style={{
          backgroundColor: "#161E19",
          border: "1px solid #1F2A24",
          color: "#ECEFEA",
        }}
      >
        <h2
          id="session-expired-title"
          className="text-xl tracking-tight mb-2"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "var(--season-heading)",
          }}
        >
          Session expired
        </h2>
        <p className="text-sm leading-relaxed mb-6" style={{ color: "#8B9A8E" }}>
          For your security, you&rsquo;ve been signed out. Sign in again to
          keep planning.
        </p>
        <button
          type="button"
          onClick={onSignIn}
          className="season-transition inline-flex w-full items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide hover:-translate-y-0.5 active:translate-y-0"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
            boxShadow: "0 8px 24px var(--season-button-shadow)",
          }}
        >
          <LogIn className="size-4" />
          Sign in again
        </button>
      </div>
    </div>
  );
}
