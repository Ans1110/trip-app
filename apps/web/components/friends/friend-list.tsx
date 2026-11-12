"use client";

import {
  CalendarDays,
  Loader2,
  MessageCircle,
  ShieldOff,
  Trash2,
  UserPlus2,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { useChatWindowStore } from "@/store/chat-window-store";

import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
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
import type { Friend, UserSummary } from "@/lib/apis/friend-api";

export function FriendList({ onNavigate }: { onNavigate?: () => void }) {
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
        renderItem={(f) => (
          <FriendRow key={f.user.id} friend={f} onNavigate={onNavigate} />
        )}
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
      <h3 className="text-sm font-medium" style={{ color: "var(--muted-foreground)" }}>
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
            <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
              Searching…
            </p>
          )}
          {searchQuery.isError && (
            <p className="text-sm" style={{ color: "var(--destructive)" }}>
              {errorMessage(searchQuery.error)}
            </p>
          )}
          {!searchQuery.isLoading && results.length === 0 && (
            <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
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
            <p className="text-sm" style={{ color: "var(--destructive)" }}>
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
      style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
    >
      <Avatar user={user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "var(--foreground)" }}>
          {user.name || user.email}
        </p>
        <p className="text-xs truncate" style={{ color: "var(--muted-foreground)" }}>
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
          color: "var(--foreground)",
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

function FriendRow({
  friend,
  onNavigate,
}: {
  friend: Friend;
  onNavigate?: () => void;
}) {
  const router = useRouter();
  const remove = useRemoveFriend();
  const block = useBlockUser();
  const openChat = useChatWindowStore((s) => s.openWithPeer);
  const busy = remove.isPending || block.isPending;
  const displayName = friend.user.name || friend.user.email;

  const openCalendar = () => {
    router.push(`/friends/${friend.user.id}/calendar`);
    onNavigate?.();
  };

  const doChat = () => {
    openChat(friend.user.id);
    onNavigate?.();
  };

  const doBlock = () => {
    if (
      window.confirm(`Block ${displayName}? They will be removed as a friend.`)
    ) {
      block.mutate({ target_id: friend.user.id });
    }
  };

  const doRemove = () => {
    if (window.confirm(`Remove ${displayName} from your friends?`)) {
      remove.mutate(friend.user.id);
    }
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div
          className="flex items-center gap-3 px-4 py-3 rounded-lg select-none"
          style={{
            backgroundColor: "var(--card)",
            border: "1px solid var(--border)",
            opacity: busy ? 0.6 : 1,
          }}
          title="Right-click for options"
        >
          <Avatar user={friend.user} />
          <div className="flex-1 min-w-0">
            <p className="text-sm truncate" style={{ color: "var(--foreground)" }}>
              {displayName}
            </p>
            <p className="text-xs truncate" style={{ color: "var(--muted-foreground)" }}>
              {friend.user.email}
            </p>
          </div>
          {busy && (
            <Loader2
              className="size-3.5 animate-spin shrink-0"
              style={{ color: "var(--muted-foreground)" }}
            />
          )}
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onSelect={doChat}>
          <MessageCircle className="size-3.5" style={{ color: "var(--muted-foreground)" }} />
          Chat
        </ContextMenuItem>
        <ContextMenuItem onSelect={openCalendar}>
          <CalendarDays className="size-3.5" style={{ color: "var(--muted-foreground)" }} />
          Calendar
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onSelect={doBlock} disabled={busy}>
          <ShieldOff className="size-3.5" style={{ color: "var(--muted-foreground)" }} />
          Block
        </ContextMenuItem>
        <ContextMenuItem
          onSelect={doRemove}
          disabled={busy}
          style={{ color: "var(--destructive)" }}
        >
          <Trash2 className="size-3.5" />
          Remove
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
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
        color: "var(--foreground)",
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
      <h3 className="text-sm font-medium" style={{ color: "var(--muted-foreground)" }}>
        {title}
      </h3>
      {loading && (
        <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
          Loading…
        </p>
      )}
      {error && (
        <p className="text-sm" style={{ color: "var(--destructive)" }}>
          {error}
        </p>
      )}
      {!loading && !error && items.length === 0 && (
        <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
          {empty}
        </p>
      )}
      <div className="flex flex-col gap-2">{items.map(renderItem)}</div>
    </section>
  );
}
