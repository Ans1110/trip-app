"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

import type { CalendarEvent, EventType } from "@/lib/calendar-api";

const EVENT_TYPE_ACCENT: Record<EventType, string> = {
  general: "#8B9A8E",
  trip: "#A8E0B4",
  flight: "#7FB3D5",
  hotel: "#E8D7B8",
  meeting: "#C3A6E0",
};

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

type WeekBar = {
  event: CalendarEvent;
  startCol: number;
  span: number;
  continuesFromPrev: boolean;
  continuesToNext: boolean;
};

type EventCalendarGridProps = {
  events: CalendarEvent[];
  isLoading?: boolean;
  errorMessage?: string;
  onDayClick?: (day: Date) => void;
  onEventClick?: (event: CalendarEvent) => void;
  onRangeChange?: (from: Date, to: Date) => void;
  userNamesById?: Record<string, string>;
};

export function EventCalendarGrid({
  events,
  isLoading,
  errorMessage,
  onDayClick,
  onEventClick,
  onRangeChange,
  userNamesById,
}: EventCalendarGridProps) {
  const [cursor, setCursor] = useState<Date | null>(null);
  const [today, setToday] = useState<Date | null>(null);
  const onRangeChangeRef = useRef(onRangeChange);

  useEffect(() => {
    onRangeChangeRef.current = onRangeChange;
  }, [onRangeChange]);

  useEffect(() => {
    const now = new Date();
    // eslint-disable-next-line react-hooks/set-state-in-effect -- one-time client-only init to avoid hydration mismatch on new Date()
    setCursor(firstOfMonth(now));
    setToday(now);
  }, []);

  const gridStart = useMemo(
    () => (cursor ? startOfGrid(cursor) : null),
    [cursor],
  );

  useEffect(() => {
    if (!gridStart) return;
    onRangeChangeRef.current?.(gridStart, addDays(gridStart, 42));
  }, [gridStart]);

  const weeks = useMemo(() => {
    if (!gridStart) return [];
    const out: Date[][] = [];
    for (let w = 0; w < 6; w++) {
      const row: Date[] = [];
      for (let d = 0; d < 7; d++) row.push(addDays(gridStart, w * 7 + d));
      out.push(row);
    }
    return out;
  }, [gridStart]);

  if (!cursor || !today) return <GridSkeleton />;

  const currentMonth = cursor.getMonth();

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => setCursor((c) => (c ? addMonths(c, -1) : c))}
            aria-label="Previous month"
            className="inline-flex items-center justify-center size-9 rounded-full hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            <ChevronLeft className="size-4" />
          </button>
          <button
            type="button"
            onClick={() => setCursor(firstOfMonth(new Date()))}
            className="px-3 py-1.5 text-xs rounded-full hover:bg-white/5"
            style={{ color: "#8B9A8E", border: "1px solid #1F2A24" }}
          >
            Today
          </button>
          <button
            type="button"
            onClick={() => setCursor((c) => (c ? addMonths(c, 1) : c))}
            aria-label="Next month"
            className="inline-flex items-center justify-center size-9 rounded-full hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            <ChevronRight className="size-4" />
          </button>
        </div>
        <span
          className="text-2xl tracking-tight"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "#ECEFEA",
          }}
        >
          {formatMonth(cursor)}
        </span>
        <div className="w-[132px]" aria-hidden />
      </div>

      {errorMessage && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage}
        </p>
      )}
      {isLoading && !errorMessage && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}

      <div
        className="rounded-2xl overflow-hidden"
        style={{ backgroundColor: "#0E1411", border: "1px solid #1F2A24" }}
      >
        <div
          className="grid grid-cols-7"
          style={{ borderBottom: "1px solid #1F2A24" }}
        >
          {WEEKDAYS.map((d) => (
            <div
              key={d}
              className="px-2 py-2 text-[11px] uppercase tracking-widest text-center"
              style={{ color: "#6B7A6F" }}
            >
              {d}
            </div>
          ))}
        </div>

        {weeks.map((week, wi) => {
          const bars = barsForWeek(events, week[0]);
          const isLastWeek = wi === weeks.length - 1;
          return (
            <div
              key={wi}
              className="relative"
              style={{
                borderBottom: isLastWeek ? undefined : "1px solid #1F2A24",
              }}
            >
              <div className="absolute inset-0 grid grid-cols-7">
                {week.map((day, di) => {
                  const inMonth = day.getMonth() === currentMonth;
                  const isToday = isSameDay(day, today);
                  const isLastCol = di === 6;
                  return (
                    <div
                      key={di}
                      className="relative"
                      style={{
                        backgroundColor: inMonth ? "#121814" : "#0B100D",
                        borderRight: isLastCol ? undefined : "1px solid #1F2A24",
                        opacity: inMonth ? 1 : 0.55,
                      }}
                    >
                      {onDayClick ? (
                        <button
                          type="button"
                          onClick={() => onDayClick(day)}
                          aria-label={`Add event on ${formatDayFull(day)}`}
                          className="absolute inset-0 hover:bg-white/[0.02] transition-colors"
                        />
                      ) : null}
                      <div className="relative px-1.5 pt-1.5 flex justify-start pointer-events-none">
                        <span
                          className="text-xs inline-flex items-center justify-center"
                          style={{
                            color: isToday ? "#0B100D" : "#ECEFEA",
                            backgroundColor: isToday
                              ? "var(--season-button)"
                              : "transparent",
                            borderRadius: 9999,
                            width: 22,
                            height: 22,
                            fontWeight: isToday ? 600 : 400,
                          }}
                        >
                          {day.getDate()}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>

              <div
                className="relative grid grid-cols-7 gap-y-1 pt-9 pb-1.5 px-1 pointer-events-none"
                style={{ minHeight: 128, gridAutoFlow: "row dense" }}
              >
                {bars.map((bar, bi) => (
                  <div
                    key={`${wi}-${bar.event.id}-${bi}`}
                    className="pointer-events-auto min-w-0 px-0.5"
                    style={{ gridColumn: `${bar.startCol} / span ${bar.span}` }}
                  >
                    <EventBar
                      bar={bar}
                      userName={userNamesById?.[bar.event.created_by]}
                      onClick={
                        onEventClick ? () => onEventClick(bar.event) : undefined
                      }
                    />
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function GridSkeleton() {
  return (
    <div
      className="rounded-2xl"
      style={{
        backgroundColor: "#0E1411",
        border: "1px solid #1F2A24",
        minHeight: 480,
      }}
    />
  );
}

function EventBar({
  bar,
  userName,
  onClick,
}: {
  bar: WeekBar;
  userName?: string;
  onClick?: () => void;
}) {
  const { event, continuesFromPrev, continuesToNext } = bar;
  const accent = event.color?.startsWith("#")
    ? event.color
    : EVENT_TYPE_ACCENT[event.event_type];
  const isTrip = event.source_type === "trip";
  const rounded = 4;
  const style: React.CSSProperties = {
    backgroundColor: `color-mix(in srgb, ${accent} 24%, transparent)`,
    borderTop: `1px solid color-mix(in srgb, ${accent} 45%, transparent)`,
    borderBottom: `1px solid color-mix(in srgb, ${accent} 45%, transparent)`,
    borderLeft: continuesFromPrev
      ? undefined
      : `1px solid color-mix(in srgb, ${accent} 45%, transparent)`,
    borderRight: continuesToNext
      ? undefined
      : `1px solid color-mix(in srgb, ${accent} 45%, transparent)`,
    borderTopLeftRadius: continuesFromPrev ? 0 : rounded,
    borderBottomLeftRadius: continuesFromPrev ? 0 : rounded,
    borderTopRightRadius: continuesToNext ? 0 : rounded,
    borderBottomRightRadius: continuesToNext ? 0 : rounded,
  };
  const content = (
    <>
      <span
        className="inline-block size-1.5 rounded-full shrink-0"
        style={{ backgroundColor: accent }}
        aria-hidden
      />
      <span
        className="text-[11px] truncate flex-1 min-w-0"
        style={{ color: "#ECEFEA" }}
      >
        {!continuesFromPrev && !event.all_day
          ? `${formatBarTime(event)} `
          : ""}
        {userName && !continuesFromPrev && (
          <span style={{ color: "#8B9A8E" }}>{userName} · </span>
        )}
        {event.title}
      </span>
      {isTrip && !continuesFromPrev && (
        <span
          className="text-[9px] uppercase tracking-wider px-1 py-0.5 rounded shrink-0 font-semibold"
          style={{ color: "#0B100D", backgroundColor: "#A8E0B4" }}
        >
          Trip
        </span>
      )}
    </>
  );
  if (!onClick) {
    return (
      <div
        title={event.title}
        className="w-full text-left px-1.5 py-1 flex items-center gap-1.5"
        style={style}
      >
        {content}
      </div>
    );
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      title={event.title}
      className="w-full text-left px-1.5 py-1 flex items-center gap-1.5 hover:brightness-125 season-transition"
      style={style}
    >
      {content}
    </button>
  );
}

function barsForWeek(events: CalendarEvent[], weekStart: Date): WeekBar[] {
  const weekEnd = addDays(weekStart, 7);
  const items: WeekBar[] = [];
  for (const ev of events) {
    const s = new Date(ev.start_at);
    const e = new Date(ev.end_at);
    if (Number.isNaN(s.getTime()) || Number.isNaN(e.getTime())) continue;
    const sDay = startOfDay(s);
    const eDay = startOfDay(e);
    if (eDay < weekStart) continue;
    if (sDay >= weekEnd) continue;
    const startIdx = Math.max(0, dayDiff(weekStart, sDay));
    const endIdx = Math.min(6, dayDiff(weekStart, eDay));
    items.push({
      event: ev,
      startCol: startIdx + 1,
      span: endIdx - startIdx + 1,
      continuesFromPrev: sDay < weekStart,
      continuesToNext: eDay >= weekEnd,
    });
  }
  items.sort((a, b) => {
    if (a.startCol !== b.startCol) return a.startCol - b.startCol;
    if (a.span !== b.span) return b.span - a.span;
    return a.event.start_at.localeCompare(b.event.start_at);
  });
  return items;
}

function firstOfMonth(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), 1);
}

function addMonths(d: Date, delta: number): Date {
  return new Date(d.getFullYear(), d.getMonth() + delta, 1);
}

function addDays(d: Date, delta: number): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() + delta);
}

function startOfGrid(cursor: Date): Date {
  const first = firstOfMonth(cursor);
  return addDays(first, -first.getDay());
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function dayDiff(a: Date, b: Date): number {
  return Math.round((b.getTime() - a.getTime()) / 86400000);
}

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function formatMonth(d: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "long",
    year: "numeric",
  }).format(d);
}

function formatDayFull(d: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  }).format(d);
}

function formatBarTime(ev: CalendarEvent): string {
  const d = new Date(ev.start_at);
  if (Number.isNaN(d.getTime())) return "";
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function pad(n: number): string {
  return n.toString().padStart(2, "0");
}
