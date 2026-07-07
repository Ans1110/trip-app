"use client";

import { useMemo, useState } from "react";

import { EventCalendarGrid } from "@/components/calendar/event-calendar-grid";
import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useFriendEvents,
  useMyEvents,
} from "@/hooks/calendar-hooks";
import { useFriends } from "@/hooks/friend-hooks";
import type { CalendarEvent } from "@/lib/calendar-api";

export function FriendCalendarView({ friendId }: { friendId: string }) {
  const [range, setRange] = useState<{ from: string; to: string }>({
    from: "",
    to: "",
  });

  const meQuery = useMe();
  const friends = useFriends();
  const myEvents = useMyEvents(range, { enabled: !!range.from });
  const friendEvents = useFriendEvents(friendId, range, {
    enabled: !!range.from,
  });

  const friend = useMemo(
    () => (friends.data ?? []).find((f) => f.user.id === friendId),
    [friends.data, friendId],
  );
  const displayName = friend?.user.name || friend?.user.email || "Friend";

  const [showAll, setShowAll] = useState(true);

  const events = useMemo(() => {
    const seen = new Map<string, CalendarEvent>();
    for (const ev of myEvents.data ?? []) seen.set(ev.id, ev);
    if (showAll) {
      for (const ev of friendEvents.data ?? [])
        if (!seen.has(ev.id)) seen.set(ev.id, ev);
    }
    return Array.from(seen.values());
  }, [myEvents.data, friendEvents.data, showAll]);

  const userNamesById = useMemo(() => {
    const map: Record<string, string> = {};
    if (meQuery.data) map[meQuery.data.id] = "You";
    if (friend) map[friend.user.id] = displayName;
    return map;
  }, [meQuery.data, friend, displayName]);

  const anyLoading = myEvents.isLoading || friendEvents.isLoading;
  const firstError = myEvents.error ?? friendEvents.error;

  return (
    <section className="flex flex-col gap-6">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            {displayName}&rsquo;s calendar
          </h1>
          <p className="text-sm mt-1" style={{ color: "#8B9A8E" }}>
            Your private &amp; friends events, alongside events{" "}
            {displayName} has shared with friends.
          </p>
        </div>
        <VisibilityToggle
          showAll={showAll}
          onChange={setShowAll}
          otherLabel={displayName}
        />
      </header>
      <EventCalendarGrid
        events={events}
        isLoading={anyLoading}
        errorMessage={
          firstError ? errorMessage(firstError) ?? undefined : undefined
        }
        onRangeChange={(from, to) =>
          setRange({ from: from.toISOString(), to: to.toISOString() })
        }
        userNamesById={userNamesById}
      />
    </section>
  );
}

function VisibilityToggle({
  showAll,
  onChange,
  otherLabel,
}: {
  showAll: boolean;
  onChange: (v: boolean) => void;
  otherLabel: string;
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
        label={`You + ${otherLabel}`}
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
