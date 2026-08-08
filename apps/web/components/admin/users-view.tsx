"use client";

import { useState } from "react";
import { CheckCircle2, Loader2, ShieldOff, UserCheck, UserX } from "lucide-react";

import {
  errorMessage,
  useAdminUsers,
  useBlockUser,
  useDeactivateUser,
  useReactivateUser,
  useUnblockUser,
} from "@/hooks/admin-hooks";
import type {
  AdminUser,
  AdminUserStatus,
  ListAdminUsersQuery,
} from "@/lib/apis/admin-api";

const STATUS_OPTIONS: { label: string; value: AdminUserStatus | "" }[] = [
  { label: "All", value: "" },
  { label: "Active", value: "active" },
  { label: "Deactivated", value: "deactivated" },
  { label: "Deleted", value: "deleted" },
];

export function UsersView() {
  const [query, setQuery] = useState<Omit<ListAdminUsersQuery, "cursor">>({
    limit: 20,
  });
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useAdminUsers(query);

  if (isLoading) return <LoadingState label="Loading users" />;
  if (error) return <ErrorState message={errorMessage(error)} />;

  const users = data?.pages.flatMap((p) => p.users) ?? [];

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
              status: v === "" ? undefined : (v as AdminUserStatus),
            }))
          }
        />
        <BlockedFilter
          value={query.is_blocked}
          onChange={(v) => setQuery((prev) => ({ ...prev, is_blocked: v }))}
        />
      </div>

      {users.length === 0 ? (
        <EmptyState message="No users match this filter." />
      ) : (
        <ul className="flex flex-col gap-3">
          {users.map((u) => (
            <UserRow key={u.id} user={u} />
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

function UserRow({ user }: { user: AdminUser }) {
  const block = useBlockUser();
  const unblock = useUnblockUser();
  const deactivate = useDeactivateUser();
  const reactivate = useReactivateUser();

  const isBusy =
    block.isPending ||
    unblock.isPending ||
    deactivate.isPending ||
    reactivate.isPending;

  return (
    <li
      className="flex flex-col md:flex-row md:items-center gap-3 p-4 rounded-xl border"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      <Avatar url={user.avatar_url} name={user.name || user.email} />
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <p className="font-medium truncate" style={{ color: "#ECEFEA" }}>
            {user.name || user.email}
          </p>
          <StatusBadge status={user.status} blocked={user.is_blocked} />
          {user.roles?.includes("admin") && <Pill label="admin" tone="accent" />}
          {user.is_verified && (
            <CheckCircle2 className="size-3.5" style={{ color: "#7FB68A" }} />
          )}
        </div>
        <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
          {user.email}
        </p>
        <p className="text-[11px] mt-1" style={{ color: "#6B7A6F" }}>
          {user.post_count} posts · {user.reported_count} reports · joined{" "}
          {new Date(user.created_at).toLocaleDateString()}
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {user.is_blocked ? (
          <ActionButton
            label="Unblock"
            onClick={() => unblock.mutate(user.id)}
            pending={unblock.isPending}
            disabled={isBusy}
          />
        ) : (
          <ActionButton
            label="Block"
            danger
            icon={<ShieldOff className="size-3.5" />}
            onClick={() => block.mutate(user.id)}
            pending={block.isPending}
            disabled={isBusy}
          />
        )}
        {user.status === "active" && (
          <ActionButton
            label="Deactivate"
            danger
            icon={<UserX className="size-3.5" />}
            onClick={() => {
              if (confirm(`Deactivate ${user.email}?`)) {
                deactivate.mutate(user.id);
              }
            }}
            pending={deactivate.isPending}
            disabled={isBusy}
          />
        )}
        {user.status === "deactivated" && (
          <ActionButton
            label="Reactivate"
            icon={<UserCheck className="size-3.5" />}
            onClick={() => reactivate.mutate(user.id)}
            pending={reactivate.isPending}
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
        placeholder="Search email or name"
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

function BlockedFilter({
  value,
  onChange,
}: {
  value: boolean | undefined;
  onChange: (v: boolean | undefined) => void;
}) {
  const current = value === undefined ? "any" : value ? "blocked" : "unblocked";
  return (
    <select
      value={current}
      onChange={(e) => {
        const v = e.target.value;
        onChange(v === "any" ? undefined : v === "blocked");
      }}
      className="px-3 py-2 text-sm rounded-lg border bg-transparent"
      style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
    >
      <option value="any" className="bg-[#121814]">
        Any
      </option>
      <option value="blocked" className="bg-[#121814]">
        Blocked
      </option>
      <option value="unblocked" className="bg-[#121814]">
        Unblocked
      </option>
    </select>
  );
}

function StatusBadge({
  status,
  blocked,
}: {
  status: AdminUserStatus;
  blocked: boolean;
}) {
  if (blocked) return <Pill label="blocked" tone="danger" />;
  if (status === "active") return <Pill label="active" tone="ok" />;
  if (status === "deactivated") return <Pill label="deactivated" tone="warn" />;
  return <Pill label={status} tone="muted" />;
}

function Pill({
  label,
  tone,
}: {
  label: string;
  tone: "ok" | "warn" | "danger" | "muted" | "accent";
}) {
  const styles: Record<typeof tone, { bg: string; fg: string }> = {
    ok: { bg: "rgba(127,182,138,0.12)", fg: "#7FB68A" },
    warn: { bg: "rgba(250,204,21,0.12)", fg: "#FACC15" },
    danger: { bg: "rgba(220,38,38,0.12)", fg: "#FCA5A5" },
    muted: { bg: "rgba(139,154,142,0.12)", fg: "#8B9A8E" },
    accent: { bg: "rgba(127,182,138,0.18)", fg: "var(--season-button)" },
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

function Avatar({ url, name }: { url?: string; name: string }) {
  return (
    <div
      className="size-10 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
      style={{
        backgroundColor: url
          ? "transparent"
          : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
        color: "#ECEFEA",
      }}
    >
      {url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={url} alt="" className="size-full object-cover" />
      ) : (
        <span aria-hidden>{name.trim()[0]?.toUpperCase() || "?"}</span>
      )}
    </div>
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
