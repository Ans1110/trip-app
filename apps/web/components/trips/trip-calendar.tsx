"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueries } from "@tanstack/react-query";

import { useMe } from "@/hooks/auth-hooks";
import {
  calendarKeys,
  errorMessage,
  useMyEvents,
} from "@/hooks/calendar-hooks";
import { useRoom } from "@/hooks/trip-hooks";
import { calendarApi, type CalendarEvent } from "@/lib/apis/calendar-api";
import { ApiError } from "@/lib/utils";

import { EventCalendarGrid } from "@/components/calendar/event-calendar-grid";

export function TripCalendar({ tripId }: { tripId: string }) {
  const router = useRouter();
  const meQuery = useMe();
  const roomQuery = useRoom(tripId);
  const myId = meQuery.data?.id;

  const [range, setRange] = useState<{ from: string; to: string }>({
    from: "",
    to: "",
  });

  const myEvents = useMyEvents(range, { enabled: !!range.from });

  const otherMemberIds = useMemo(() => {
    if (!roomQuery.data || !myId) return [];
    return roomQuery.data.members
      .map((m) => m.user.id)
      .filter((id) => id !== myId);
  }, [roomQuery.data, myId]);

  const userNamesById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const m of roomQuery.data?.members ?? []) {
      map[m.user.id] = m.user.name || m.user.email;
    }
    if (myId) map[myId] = "You";
    return map;
  }, [roomQuery.data, myId]);

  const memberQueries = useQueries({
    queries: otherMemberIds.map((memberId) => ({
      queryKey: calendarKeys.friend(memberId, range),
      queryFn: ({ signal }: { signal?: AbortSignal }) =>
        calendarApi.listFriend(memberId, range, signal),
      enabled: !!range.from,
    })),
  });

  const [showAll, setShowAll] = useState(true);

  const events = useMemo(() => {
    const seen = new Map<string, CalendarEvent>();
    for (const ev of myEvents.data ?? []) seen.set(ev.id, ev);
    if (showAll) {
      for (const q of memberQueries) {
        for (const ev of q.data ?? [])
          if (!seen.has(ev.id)) seen.set(ev.id, ev);
      }
    }
    return Array.from(seen.values());
  }, [myEvents.data, memberQueries, showAll]);

  const anyLoading =
    myEvents.isLoading || memberQueries.some((q) => q.isLoading);
  const firstError =
    myEvents.error ??
    (memberQueries.find((q) => q.isError)?.error as ApiError | undefined);

  const onRangeChange = (from: Date, to: Date) =>
    setRange({ from: from.toISOString(), to: to.toISOString() });

  const onEventClick = (event: CalendarEvent) => {
    if (event.source_type === "trip" && event.source_id) {
      router.push(`/trips/${event.source_id}`);
    }
  };

  if (roomQuery.isLoading) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        Loading…
      </p>
    );
  }

  if (roomQuery.isError) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(roomQuery.error) ?? "Room not found."}
      </p>
    );
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2
            className="text-xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            Shared calendar
          </h2>
          <p className="text-xs mt-1" style={{ color: "#8B9A8E" }}>
            Your private &amp; friends events, alongside every room
            member&rsquo;s friends-visible events.
          </p>
        </div>
        <VisibilityToggle showAll={showAll} onChange={setShowAll} />
      </div>
      <EventCalendarGrid
        events={events}
        isLoading={anyLoading}
        errorMessage={
          firstError ? (errorMessage(firstError) ?? undefined) : undefined
        }
        onRangeChange={onRangeChange}
        onEventClick={onEventClick}
        userNamesById={userNamesById}
      />
    </section>
  );
}

function VisibilityToggle({
  showAll,
  onChange,
}: {
  showAll: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <div
      role="group"
      aria-label="Filter events"
      className="inline-flex items-center gap-0.5 p-0.5 rounded-full shrink-0"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <ToggleBtn
        active={showAll}
        onClick={() => onChange(true)}
        label="Everyone"
      />
      <ToggleBtn
        active={!showAll}
        onClick={() => onChange(false)}
        label="Just you"
      />
    </div>
  );
}

function ToggleBtn({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className="season-transition px-3 py-1 rounded-full text-xs font-medium"
      style={{
        backgroundColor: active
          ? "color-mix(in srgb, var(--season-button) 22%, transparent)"
          : "transparent",
        color: active ? "#ECEFEA" : "#8B9A8E",
      }}
    >
      {label}
    </button>
  );
}
