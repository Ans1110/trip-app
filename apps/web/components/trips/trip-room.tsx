"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Copy, Loader2, LogOut, RefreshCcw, Trash2 } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useLeaveRoom,
  useRegenerateCode,
  useRemoveMember,
  useRoom,
} from "@/hooks/trip-hooks";
import type { RoomMember, UserSummary } from "@/lib/apis/trip-api";

import { RoomQr } from "./room-qr";

export function TripRoom({
  tripId,
  canManage,
}: {
  tripId: string;
  canManage: boolean;
}) {
  const router = useRouter();
  const roomQuery = useRoom(tripId);
  const meQuery = useMe();
  const regen = useRegenerateCode();
  const leave = useLeaveRoom();
  const removeMember = useRemoveMember();
  const [copied, setCopied] = useState(false);

  const room = roomQuery.data;
  const myId = meQuery.data?.id;

  const handleCopy = async () => {
    if (!room) return;
    try {
      await navigator.clipboard.writeText(room.room_code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  const handleLeave = () => {
    if (!window.confirm("Leave this trip's room?")) return;
    leave.mutate(tripId, { onSuccess: () => router.replace("/trips") });
  };

  if (roomQuery.isLoading) {
    return (
      <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
        Loading…
      </p>
    );
  }

  if (roomQuery.isError || !room) {
    return (
      <p className="text-sm" style={{ color: "var(--destructive)" }}>
        {errorMessage(roomQuery.error) ?? "Room not found."}
      </p>
    );
  }

  return (
    <section className="flex flex-col gap-6">
      <div
        className="rounded-2xl p-5 grid grid-cols-1 md:grid-cols-[auto_1fr] gap-6 items-center"
        style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
      >
        <RoomQr code={room.room_code} />
        <div className="flex flex-col gap-3">
          <p
            className="text-xs font-medium tracking-wider uppercase"
            style={{ color: "var(--muted-foreground)" }}
          >
            Room code
          </p>
          <code
            className="text-2xl tracking-[0.3em] px-4 py-2 rounded-lg self-start"
            style={{
              fontFamily: "var(--font-mono, monospace)",
              backgroundColor: "var(--background)",
              border: "1px solid var(--border)",
              color: "var(--foreground)",
            }}
          >
            {room.room_code}
          </code>
          <p className="text-xs" style={{ color: "var(--muted-foreground)" }}>
            Share this code or have a friend scan the QR to join the trip.
          </p>
          <div className="flex items-center gap-2 flex-wrap">
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-muted"
              style={{ color: "var(--foreground)", border: "1px solid var(--border)" }}
            >
              <Copy className="size-3.5" />
              {copied ? "Copied!" : "Copy code"}
            </button>
            {canManage && (
              <button
                type="button"
                onClick={() => regen.mutate(tripId)}
                disabled={regen.isPending}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-muted disabled:opacity-60"
                style={{ color: "var(--foreground)", border: "1px solid var(--border)" }}
              >
                {regen.isPending ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : (
                  <RefreshCcw className="size-3.5" />
                )}
                Regenerate
              </button>
            )}
          </div>
          {regen.isError && (
            <p className="text-xs" style={{ color: "var(--destructive)" }}>
              {errorMessage(regen.error)}
            </p>
          )}
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-medium" style={{ color: "var(--muted-foreground)" }}>
          Members
        </h3>
        <div className="flex flex-col gap-2">
          {room.members.map((m) => (
            <MemberRow
              key={m.user.id}
              member={m}
              isSelf={m.user.id === myId}
              canManage={canManage}
              onRemove={() =>
                removeMember.mutate({ id: tripId, userId: m.user.id })
              }
              onLeave={handleLeave}
              busy={removeMember.isPending || leave.isPending}
            />
          ))}
        </div>
      </div>
    </section>
  );
}

function MemberRow({
  member,
  isSelf,
  canManage,
  onRemove,
  onLeave,
  busy,
}: {
  member: RoomMember;
  isSelf: boolean;
  canManage: boolean;
  onRemove: () => void;
  onLeave: () => void;
  busy: boolean;
}) {
  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "var(--card)", border: "1px solid var(--border)" }}
    >
      <Avatar user={member.user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "var(--foreground)" }}>
          {member.user.name || member.user.email}
          {isSelf && (
            <span className="ml-2 text-xs" style={{ color: "var(--muted-foreground)" }}>
              you
            </span>
          )}
        </p>
        <p className="text-xs truncate" style={{ color: "var(--muted-foreground)" }}>
          {member.role}
        </p>
      </div>
      {isSelf ? (
        <button
          type="button"
          onClick={onLeave}
          disabled={busy}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-muted disabled:opacity-60"
          style={{ color: "var(--destructive)" }}
        >
          <LogOut className="size-3.5" />
          Leave
        </button>
      ) : (
        canManage &&
        member.role !== "admin" && (
          <button
            type="button"
            onClick={() => {
              if (
                window.confirm(
                  `Remove ${member.user.name || member.user.email}?`,
                )
              ) {
                onRemove();
              }
            }}
            disabled={busy}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-muted disabled:opacity-60"
            style={{ color: "var(--destructive)" }}
          >
            <Trash2 className="size-3.5" />
            Remove
          </button>
        )
      )}
    </div>
  );
}

function Avatar({ user }: { user: UserSummary }) {
  const initial = (user.name || user.email || "?").charAt(0).toUpperCase();
  return (
    <span
      className="size-9 rounded-full inline-flex items-center justify-center shrink-0 text-sm font-medium overflow-hidden"
      style={{
        backgroundColor: user.avatar_url
          ? "transparent"
          : "color-mix(in srgb, var(--season-button) 22%, transparent)",
        color: "var(--foreground)",
      }}
    >
      {user.avatar_url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={user.avatar_url}
          alt=""
          className="size-full object-cover"
        />
      ) : (
        <span aria-hidden>{initial}</span>
      )}
    </span>
  );
}
