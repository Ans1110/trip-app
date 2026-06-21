"use client";

import { Loader2, ShieldOff, Trash2, UserPlus2 } from "lucide-react";
import { useState } from "react";

import { Input } from "@/components/ui/input";
import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useBlockUser,
  useFriends,
  useRemoveFriend,
  useSearchUsers,
  useSendRequest,
} from "@/hooks/friend-hooks";
import type { Friend, UserSummary } from "@/lib/friend-api";

export function FriendList() {
  const [query, setQuery] = useState("");
  const friendsQuery = useFriends();

  return (
    <div className="flex flex-col gap-8">
      <SearchPanel query={query} setQuery={setQuery} />

      <Section
        title="Your friends"
        empty="No friends yet. Find someone above."
        loading={friendsQuery.isLoading}
        error={errorMessage(friendsQuery.error)}
        items={friendsQuery.data ?? []}
        renderItem={(f) => <FriendRow key={f.user.id} friend={f} />}
      />
    </div>
  );
}

function SearchPanel({
  query,
  setQuery,
}: {
  query: string;
  setQuery: (v: string) => void;
}) {
  const trimmed = query.trim();
  const searchQuery = useSearchUsers(trimmed, 10);
  const sendRequest = useSendRequest();
  const meQuery = useMe();
  const myId = meQuery.data?.id ?? null;
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [sentIds, setSentIds] = useState<Set<string>>(new Set());

  const handleSend = (id: string) => {
    if (id === myId) return;
    setPendingId(id);
    sendRequest.mutate(
      { receiver_id: id },
      {
        onSuccess: () => {
          setSentIds((prev) => new Set(prev).add(id));
        },
        onSettled: () => setPendingId(null),
      },
    );
  };

  const results = (searchQuery.data ?? []).filter((u) => u.id !== myId);
  const showResults = trimmed.length > 0;

  return (
    <section className="flex flex-col gap-3">
      <h3 className="text-sm font-medium" style={{ color: "#8B9A8E" }}>
        Find people
      </h3>
      <Input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search by name or email"
        className="px-4 py-3"
      />
      {showResults && (
        <div className="flex flex-col gap-2 mt-1">
          {searchQuery.isLoading && (
            <p className="text-sm" style={{ color: "#8B9A8E" }}>
              Searching…
            </p>
          )}
          {searchQuery.isError && (
            <p className="text-sm" style={{ color: "#FCA5A5" }}>
              {errorMessage(searchQuery.error)}
            </p>
          )}
          {!searchQuery.isLoading && results.length === 0 && (
            <p className="text-sm" style={{ color: "#8B9A8E" }}>
              No matches for &ldquo;{trimmed}&rdquo;.
            </p>
          )}
          {results.map((u) => (
            <SearchResultRow
              key={u.id}
              user={u}
              sent={sentIds.has(u.id)}
              pending={pendingId === u.id}
              onSend={() => handleSend(u.id)}
            />
          ))}
          {sendRequest.isError && pendingId === null && (
            <p className="text-sm" style={{ color: "#FCA5A5" }}>
              {errorMessage(sendRequest.error)}
            </p>
          )}
        </div>
      )}
    </section>
  );
}

function SearchResultRow({
  user,
  sent,
  pending,
  onSend,
}: {
  user: UserSummary;
  sent: boolean;
  pending: boolean;
  onSend: () => void;
}) {
  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Avatar user={user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
          {user.name || user.email}
        </p>
        <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
          {user.email}
        </p>
      </div>
      <button
        type="button"
        onClick={onSend}
        disabled={sent || pending}
        className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full disabled:opacity-60"
        style={{
          backgroundColor:
            "color-mix(in srgb, var(--season-button) 22%, transparent)",
          color: "#ECEFEA",
          border:
            "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)",
        }}
      >
        {pending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <UserPlus2 className="size-3.5" />
        )}
        {sent ? "Sent" : "Add"}
      </button>
    </div>
  );
}

function FriendRow({ friend }: { friend: Friend }) {
  const remove = useRemoveFriend();
  const block = useBlockUser();
  const busy = remove.isPending || block.isPending;
  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Avatar user={friend.user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
          {friend.user.name || friend.user.email}
        </p>
        <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
          {friend.user.email}
        </p>
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        <button
          type="button"
          onClick={() => {
            if (window.confirm(`Block ${friend.user.name || friend.user.email}? They will be removed as a friend.`)) {
              block.mutate({ target_id: friend.user.id });
            }
          }}
          disabled={busy}
          aria-label="Block"
          title="Block"
          className="season-transition inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5 disabled:opacity-60"
          style={{ color: "#8B9A8E" }}
        >
          {block.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <ShieldOff className="size-3.5" />
          )}
        </button>
        <button
          type="button"
          onClick={() => remove.mutate(friend.user.id)}
          disabled={busy}
          className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
          style={{ color: "#FCA5A5" }}
        >
          {remove.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Trash2 className="size-3.5" />
          )}
          Remove
        </button>
      </div>
    </div>
  );
}

export function Avatar({ user }: { user: UserSummary }) {
  const initial = (user.name || user.email || "?").charAt(0).toUpperCase();
  return user.avatar_url ? (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={user.avatar_url}
      alt=""
      className="size-9 rounded-full object-cover shrink-0"
    />
  ) : (
    <div
      className="size-9 rounded-full flex items-center justify-center shrink-0 text-sm font-medium"
      style={{
        backgroundColor:
          "color-mix(in srgb, var(--season-button) 22%, transparent)",
        color: "#ECEFEA",
      }}
    >
      {initial}
    </div>
  );
}

export function Section<T>({
  title,
  empty,
  loading,
  error,
  items,
  renderItem,
}: {
  title: string;
  empty: string;
  loading: boolean;
  error: string | null;
  items: T[];
  renderItem: (item: T) => React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <h3 className="text-sm font-medium" style={{ color: "#8B9A8E" }}>
        {title}
      </h3>
      {loading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {error && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      )}
      {!loading && !error && items.length === 0 && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {empty}
        </p>
      )}
      <div className="flex flex-col gap-2">{items.map(renderItem)}</div>
    </section>
  );
}
