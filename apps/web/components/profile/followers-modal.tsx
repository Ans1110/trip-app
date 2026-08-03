"use client";

import Link from "next/link";
import { useState } from "react";
import { Loader2 } from "lucide-react";

import { Modal } from "@/components/trips/modal";
import {
  errorMessage,
  useFollowers,
  useFollowing,
} from "@/hooks/profile-hooks";
import type { ProfileUserSummary } from "@/lib/apis/profile-api";

export type FollowersTab = "followers" | "following";

export function FollowersModal({
  userId,
  initialTab,
  followersCount,
  followingCount,
  onClose,
}: {
  userId: string;
  initialTab: FollowersTab;
  followersCount: number;
  followingCount: number;
  onClose: () => void;
}) {
  const [tab, setTab] = useState<FollowersTab>(initialTab);
  return (
    <Modal
      title={tab === "followers" ? "Followers" : "Following"}
      onClose={onClose}
      size="md"
    >
      <div className="flex flex-col gap-4">
        <div
          className="flex gap-1 p-1 rounded-full text-sm self-start"
          style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
        >
          <TabButton
            active={tab === "followers"}
            onClick={() => setTab("followers")}
          >
            Followers · {followersCount}
          </TabButton>
          <TabButton
            active={tab === "following"}
            onClick={() => setTab("following")}
          >
            Following · {followingCount}
          </TabButton>
        </div>
        {tab === "followers" ? (
          <FollowersList userId={userId} onClose={onClose} />
        ) : (
          <FollowingList userId={userId} onClose={onClose} />
        )}
      </div>
    </Modal>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="season-transition px-3 py-1.5 rounded-full text-xs font-medium"
      style={{
        backgroundColor: active ? "var(--season-button)" : "transparent",
        color: active ? "#0B100D" : "#ECEFEA",
      }}
    >
      {children}
    </button>
  );
}

function FollowersList({
  userId,
  onClose,
}: {
  userId: string;
  onClose: () => void;
}) {
  const { data, isLoading, error } = useFollowers(userId);
  return (
    <UserList
      users={data?.users}
      isLoading={isLoading}
      error={error}
      onClose={onClose}
      emptyText="No followers yet"
    />
  );
}

function FollowingList({
  userId,
  onClose,
}: {
  userId: string;
  onClose: () => void;
}) {
  const { data, isLoading, error } = useFollowing(userId);
  return (
    <UserList
      users={data?.users}
      isLoading={isLoading}
      error={error}
      onClose={onClose}
      emptyText="Not following anyone yet"
    />
  );
}

function UserList({
  users,
  isLoading,
  error,
  emptyText,
  onClose,
}: {
  users?: ProfileUserSummary[];
  isLoading: boolean;
  error: unknown;
  emptyText: string;
  onClose: () => void;
}) {
  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm py-8 justify-center"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading
      </div>
    );
  }
  if (error) {
    return (
      <p className="text-sm py-8 text-center" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Failed to load"}
      </p>
    );
  }
  if (!users || users.length === 0) {
    return (
      <p className="text-sm py-8 text-center" style={{ color: "#8B9A8E" }}>
        {emptyText}
      </p>
    );
  }
  return (
    <ul className="flex flex-col gap-1 max-h-[420px] overflow-y-auto chat-scroll -mx-2 px-2">
      {users.map((u) => (
        <li key={u.user_id}>
          <Link
            href={`/profile/${u.username}`}
            onClick={onClose}
            className="flex items-center gap-3 p-2 rounded-xl hover:bg-white/5"
          >
            <div
              className="size-10 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
              style={{
                backgroundColor: u.avatar_url
                  ? "transparent"
                  : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
                color: "#ECEFEA",
              }}
            >
              {u.avatar_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={u.avatar_url}
                  alt={u.name}
                  className="size-full object-cover"
                />
              ) : (
                <span aria-hidden>
                  {(u.username || u.name || "U").trim()[0]!.toUpperCase()}
                </span>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <p
                className="text-sm font-medium truncate"
                style={{ color: "#ECEFEA" }}
              >
                {u.name || u.username}
              </p>
              <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
                @{u.username}
              </p>
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}
