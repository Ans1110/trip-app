"use client";

import { useState } from "react";
import { ChevronLeft, ShieldOff, UserPlus2, X } from "lucide-react";

import { BlocksPanel } from "./blocks-panel";
import { FriendList } from "./friend-list";
import { RequestsPanel } from "./requests-panel";

type View = "friends" | "requests" | "blocks";

const TITLES: Record<View, string> = {
  friends: "Friends",
  requests: "Requests",
  blocks: "Blocks",
};

export function FriendsTabs({ onClose }: { onClose?: () => void }) {
  const [view, setView] = useState<View>("friends");

  return (
    <div className="flex flex-col h-full">
      <header
        className="flex items-center gap-2 px-6 py-4 border-b shrink-0"
        style={{ borderColor: "#1F2A24" }}
      >
        {view !== "friends" && (
          <button
            type="button"
            onClick={() => setView("friends")}
            aria-label="Back to friends"
            className="inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5"
            style={{ color: "#8B9A8E" }}
          >
            <ChevronLeft className="size-4" />
          </button>
        )}
        <h2
          className="flex-1 text-xl tracking-tight"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "#ECEFEA",
          }}
        >
          {TITLES[view]}
        </h2>
        {view === "friends" && (
          <>
            <IconBtn
              icon={UserPlus2}
              label="Requests"
              onClick={() => setView("requests")}
            />
            <IconBtn
              icon={ShieldOff}
              label="Blocks"
              onClick={() => setView("blocks")}
            />
          </>
        )}
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5"
            style={{ color: "#8B9A8E" }}
          >
            <X className="size-4" />
          </button>
        )}
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-5">
        {view === "friends" && <FriendList />}
        {view === "requests" && <RequestsPanel />}
        {view === "blocks" && <BlocksPanel />}
      </div>
    </div>
  );
}

function IconBtn({
  icon: Icon,
  label,
  onClick,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className="inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5"
      style={{ color: "#ECEFEA" }}
    >
      <Icon className="size-4" />
    </button>
  );
}
