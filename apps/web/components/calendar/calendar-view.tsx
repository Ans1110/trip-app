"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";

import { errorMessage, useMyEvents } from "@/hooks/calendar-hooks";
import type { CalendarEvent } from "@/lib/apis/calendar-api";

import { CreateEventDialog } from "./create-event-dialog";
import { EditEventDialog } from "./edit-event-dialog";
import { EventCalendarGrid } from "./event-calendar-grid";

export function CalendarView() {
  const router = useRouter();
  const [range, setRange] = useState<{ from: string; to: string }>({
    from: "",
    to: "",
  });
  const [createDefault, setCreateDefault] = useState<string | null>(null);
  const [openCreate, setOpenCreate] = useState(false);
  const [editing, setEditing] = useState<CalendarEvent | null>(null);

  const query = useMyEvents(range, { enabled: !!range.from });
  const events = useMemo(() => query.data ?? [], [query.data]);

  const onRangeChange = (from: Date, to: Date) =>
    setRange({ from: from.toISOString(), to: to.toISOString() });

  const openCreateAt = (day?: Date) => {
    setCreateDefault(day ? dayKey(day) : null);
    setOpenCreate(true);
  };

  const onEventClick = (event: CalendarEvent) => {
    if (event.source_type === "trip") {
      if (event.source_id) router.push(`/trips/${event.source_id}`);
      return;
    }
    setEditing(event);
  };

  return (
    <section className="flex flex-col gap-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            Calendar
          </h1>
          <p className="text-sm mt-1" style={{ color: "#8B9A8E" }}>
            Your events, plus every trip on your schedule.
          </p>
        </div>
        <button
          type="button"
          onClick={() => openCreateAt()}
          className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full"
          style={{ backgroundColor: "var(--season-button)", color: "#0B100D" }}
        >
          <Plus className="size-4" />
          New event
        </button>
      </header>

      <EventCalendarGrid
        events={events}
        isLoading={query.isLoading}
        errorMessage={
          query.isError ? (errorMessage(query.error) ?? undefined) : undefined
        }
        onRangeChange={onRangeChange}
        onDayClick={(day) => openCreateAt(day)}
        onEventClick={onEventClick}
      />

      {openCreate && (
        <CreateEventDialog
          onClose={() => {
            setOpenCreate(false);
            setCreateDefault(null);
          }}
          defaultDate={createDefault ?? undefined}
        />
      )}
      {editing && (
        <EditEventDialog event={editing} onClose={() => setEditing(null)} />
      )}
    </section>
  );
}

function pad(n: number): string {
  return n.toString().padStart(2, "0");
}

function dayKey(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
