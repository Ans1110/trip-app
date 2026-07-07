"use client";

import { Loader2, ShieldOff } from "lucide-react";

import { errorMessage, useBlocks, useUnblockUser } from "@/hooks/friend-hooks";
import type { FriendBlock } from "@/lib/apis/friend-api";

import { Avatar, Section } from "./friend-list";

export function BlocksPanel() {
  const blocks = useBlocks();
  return (
    <Section
      title="Blocked users"
      empty="You haven't blocked anyone."
      loading={blocks.isLoading}
      error={errorMessage(blocks.error)}
      items={blocks.data ?? []}
      renderItem={(b) => <BlockRow key={b.user.id} block={b} />}
    />
  );
}

function BlockRow({ block }: { block: FriendBlock }) {
  const unblock = useUnblockUser();
  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Avatar user={block.user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
          {block.user.name || block.user.email}
        </p>
        <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
          {block.user.email}
        </p>
      </div>
      <button
        type="button"
        onClick={() => unblock.mutate(block.user.id)}
        disabled={unblock.isPending}
        className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
        style={{ color: "#ECEFEA" }}
      >
        {unblock.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <ShieldOff className="size-3.5" />
        )}
        Unblock
      </button>
    </div>
  );
}
