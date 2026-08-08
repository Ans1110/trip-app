"use client";

import { useState } from "react";
import { Loader2, Trash2, Undo2 } from "lucide-react";

import {
  errorMessage,
  useAdminDeletePost,
  useAdminPosts,
  useAdminRestorePost,
} from "@/hooks/admin-hooks";
import type {
  AdminPost,
  AdminPostStatus,
  ListAdminPostsQuery,
} from "@/lib/apis/admin-api";

const STATUS_OPTIONS: { label: string; value: AdminPostStatus | "" }[] = [
  { label: "All", value: "" },
  { label: "Published", value: "published" },
  { label: "Draft", value: "draft" },
  { label: "Archived", value: "archived" },
];

export function PostsView() {
  const [query, setQuery] = useState<Omit<ListAdminPostsQuery, "cursor">>({
    limit: 20,
    include_deleted: false,
  });
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useAdminPosts(query);

  if (isLoading) return <LoadingState label="Loading posts" />;
  if (error) return <ErrorState message={errorMessage(error)} />;

  const posts = data?.pages.flatMap((p) => p.posts) ?? [];

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <SearchInput
          value={query.q ?? ""}
          onCommit={(q) => setQuery((prev) => ({ ...prev, q: q || undefined }))}
        />
        <StatusSelect
          value={query.status ?? ""}
          onChange={(v) =>
            setQuery((prev) => ({
              ...prev,
              status: v === "" ? undefined : (v as AdminPostStatus),
            }))
          }
        />
        <label
          className="inline-flex items-center gap-2 text-xs px-3 py-2 rounded-lg border cursor-pointer"
          style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
        >
          <input
            type="checkbox"
            checked={!!query.include_deleted}
            onChange={(e) =>
              setQuery((prev) => ({
                ...prev,
                include_deleted: e.target.checked,
              }))
            }
          />
          Include deleted
        </label>
      </div>

      {posts.length === 0 ? (
        <EmptyState message="No posts match this filter." />
      ) : (
        <ul className="flex flex-col gap-3">
          {posts.map((p) => (
            <PostRow key={p.id} post={p} />
          ))}
        </ul>
      )}

      {hasNextPage && (
        <div className="mt-6 flex justify-center">
          <button
            type="button"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
            className="px-4 py-2 text-sm rounded-full border hover:bg-white/5 disabled:opacity-60"
            style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
          >
            {isFetchingNextPage ? "Loading…" : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}

function PostRow({ post }: { post: AdminPost }) {
  const del = useAdminDeletePost();
  const restore = useAdminRestorePost();
  const isBusy = del.isPending || restore.isPending;

  return (
    <li
      className="flex flex-col md:flex-row md:items-start gap-3 p-4 rounded-xl border"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-2 mb-1">
          <p
            className="font-medium truncate max-w-full"
            style={{ color: "#ECEFEA" }}
          >
            {post.title || "Untitled"}
          </p>
          <StatusPill status={post.status} deleted={post.is_deleted} />
          {post.report_count > 0 && (
            <Pill label={`${post.report_count} reports`} tone="warn" />
          )}
        </div>
        <p className="text-xs mb-1" style={{ color: "#8B9A8E" }}>
          by {post.author.name || post.author.email} ·{" "}
          {new Date(post.created_at).toLocaleDateString()}
        </p>
        <p
          className="text-sm line-clamp-2 whitespace-pre-wrap"
          style={{ color: "#B7C0B8" }}
        >
          {post.content}
        </p>
        <p className="text-[11px] mt-2" style={{ color: "#6B7A6F" }}>
          {post.like_count} likes · {post.comment_count} comments
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {post.is_deleted ? (
          <ActionButton
            label="Restore"
            icon={<Undo2 className="size-3.5" />}
            onClick={() => restore.mutate(post.id)}
            pending={restore.isPending}
            disabled={isBusy}
          />
        ) : (
          <ActionButton
            label="Delete"
            danger
            icon={<Trash2 className="size-3.5" />}
            onClick={() => {
              if (confirm(`Delete post "${post.title || post.id}"?`)) {
                del.mutate(post.id);
              }
            }}
            pending={del.isPending}
            disabled={isBusy}
          />
        )}
      </div>
    </li>
  );
}

function SearchInput({
  value,
  onCommit,
}: {
  value: string;
  onCommit: (q: string) => void;
}) {
  const [text, setText] = useState(value);
  return (
    <form
      className="flex-1 min-w-[180px]"
      onSubmit={(e) => {
        e.preventDefault();
        onCommit(text.trim());
      }}
    >
      <input
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="Search title or content"
        className="w-full px-3 py-2 text-sm rounded-lg border bg-transparent outline-none focus:ring-2 focus:ring-[var(--season-accent)]/40"
        style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
      />
    </form>
  );
}

function StatusSelect({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="px-3 py-2 text-sm rounded-lg border bg-transparent"
      style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
    >
      {STATUS_OPTIONS.map((o) => (
        <option key={o.value} value={o.value} className="bg-[#121814]">
          {o.label}
        </option>
      ))}
    </select>
  );
}

function StatusPill({
  status,
  deleted,
}: {
  status: AdminPostStatus;
  deleted: boolean;
}) {
  if (deleted) return <Pill label="deleted" tone="danger" />;
  if (status === "published") return <Pill label="published" tone="ok" />;
  if (status === "draft") return <Pill label="draft" tone="muted" />;
  return <Pill label={status} tone="warn" />;
}

function Pill({
  label,
  tone,
}: {
  label: string;
  tone: "ok" | "warn" | "danger" | "muted";
}) {
  const styles: Record<typeof tone, { bg: string; fg: string }> = {
    ok: { bg: "rgba(127,182,138,0.12)", fg: "#7FB68A" },
    warn: { bg: "rgba(250,204,21,0.12)", fg: "#FACC15" },
    danger: { bg: "rgba(220,38,38,0.12)", fg: "#FCA5A5" },
    muted: { bg: "rgba(139,154,142,0.12)", fg: "#8B9A8E" },
  };
  const s = styles[tone];
  return (
    <span
      className="text-[10px] uppercase tracking-wider px-1.5 py-0.5 rounded"
      style={{ backgroundColor: s.bg, color: s.fg }}
    >
      {label}
    </span>
  );
}

function ActionButton({
  label,
  onClick,
  pending,
  disabled,
  danger,
  icon,
}: {
  label: string;
  onClick: () => void;
  pending?: boolean;
  disabled?: boolean;
  danger?: boolean;
  icon?: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs rounded-full border hover:bg-white/5 disabled:opacity-60"
      style={{
        borderColor: danger ? "rgba(220,38,38,0.35)" : "#1F2A24",
        color: danger ? "#FCA5A5" : "#ECEFEA",
      }}
    >
      {pending ? <Loader2 className="size-3.5 animate-spin" /> : icon}
      {label}
    </button>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div
      className="flex items-center gap-2 text-sm py-8"
      style={{ color: "#8B9A8E" }}
    >
      <Loader2 className="size-4 animate-spin" />
      {label}
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <p className="text-sm py-8" style={{ color: "#8B9A8E" }}>
      {message}
    </p>
  );
}

function ErrorState({ message }: { message: string | null }) {
  return (
    <p className="text-sm py-8" style={{ color: "#FCA5A5" }}>
      {message ?? "Failed to load"}
    </p>
  );
}
