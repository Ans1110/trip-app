"use client";

import { useRouter } from "next/navigation";
import { Loader2, Users } from "lucide-react";

import {
  errorMessage,
  useJoinRoom,
  useRoomPreviewByCode,
} from "@/hooks/trip-hooks";

export function RoomPreview({ code }: { code: string }) {
  const router = useRouter();
  const preview = useRoomPreviewByCode(code);
  const join = useJoinRoom();

  if (preview.isLoading) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        Loading…
      </p>
    );
  }

  if (preview.isError || !preview.data) {
    return (
      <div
        className="rounded-2xl p-6"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(preview.error) ??
            "We couldn't find a trip with that code."}
        </p>
      </div>
    );
  }

  const room = preview.data;
  const handleJoin = () => {
    join.mutate(
      { code },
      {
        onSuccess: (result) => router.replace(`/trips/${result.trip_id}`),
      },
    );
  };

  return (
    <div
      className="rounded-2xl overflow-hidden"
      style={{ backgroundColor: "#161E19", border: "1px solid #1F2A24" }}
    >
      <div
        className="h-40 w-full"
        style={{
          background: room.cover_image
            ? `url(${room.cover_image}) center / cover`
            : "linear-gradient(135deg, #1F2A24 0%, #0B100D 100%)",
        }}
        aria-hidden
      />
      <div className="px-6 py-5 flex flex-col gap-4">
        <div>
          <p
            className="text-[11px] font-medium tracking-[0.2em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            {room.status}
          </p>
          <h1
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            {room.title}
          </h1>
          {room.description && (
            <p
              className="text-sm mt-2 whitespace-pre-wrap"
              style={{ color: "#8B9A8E" }}
            >
              {room.description}
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-4 text-xs" style={{ color: "#8B9A8E" }}>
          <span>{formatRange(room.start_date, room.end_date)}</span>
          <span className="inline-flex items-center gap-1">
            <Users className="size-3.5" />
            {room.member_count}
          </span>
          <span>
            Hosted by {room.owner.name || room.owner.email}
          </span>
        </div>

        {room.already_joined ? (
          <button
            type="button"
            onClick={() => router.push(`/trips/${room.trip_id}`)}
            className="season-transition inline-flex items-center justify-center px-5 py-3 text-sm font-medium rounded-full self-start"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            Open trip
          </button>
        ) : (
          <button
            type="button"
            onClick={handleJoin}
            disabled={join.isPending}
            className="season-transition inline-flex items-center gap-2 px-5 py-3 text-sm font-medium rounded-full self-start disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {join.isPending && <Loader2 className="size-3.5 animate-spin" />}
            Join trip
          </button>
        )}
        {join.isError && (
          <p className="text-xs" style={{ color: "#FCA5A5" }}>
            {errorMessage(join.error)}
          </p>
        )}
      </div>
    </div>
  );
}

function formatRange(start: string, end: string): string {
  const fmt = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
  const startDate = new Date(start);
  const endDate = new Date(end);
  if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) {
    return `${start} – ${end}`;
  }
  return `${fmt.format(startDate)} – ${fmt.format(endDate)}`;
}
