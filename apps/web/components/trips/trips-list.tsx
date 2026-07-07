"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { Plus, UserPlus2, Users } from "lucide-react";

import { errorMessage, useTrips } from "@/hooks/trip-hooks";
import type { Trip, TripStatus } from "@/lib/apis/trip-api";

import { CreateTripDialog } from "./create-trip-dialog";
import { JoinRoomDialog } from "./join-room-dialog";

type FilterValue = "all" | TripStatus;

const filters: { value: FilterValue; label: string }[] = [
  { value: "all", label: "All" },
  { value: "planning", label: "Planning" },
  { value: "ongoing", label: "Ongoing" },
  { value: "completed", label: "Completed" },
];

export function TripsList() {
  const [filter, setFilter] = useState<FilterValue>("all");
  const [openCreate, setOpenCreate] = useState(false);
  const [openJoin, setOpenJoin] = useState(false);
  const query = useTrips(filter === "all" ? {} : { status: filter });
  const trips = query.data ?? [];

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
            Your trips
          </h1>
          <p className="text-sm mt-1" style={{ color: "#8B9A8E" }}>
            Plan, collaborate, and travel.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setOpenJoin(true)}
            className="inline-flex items-center gap-1.5 px-3 py-2 text-sm rounded-full hover:bg-white/5"
            style={{
              color: "#ECEFEA",
              border: "1px solid #1F2A24",
              backgroundColor: "#161E19",
            }}
          >
            <UserPlus2 className="size-4" />
            Join
          </button>
          <button
            type="button"
            onClick={() => setOpenCreate(true)}
            className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            <Plus className="size-4" />
            New trip
          </button>
        </div>
      </header>

      <div className="flex flex-wrap items-center gap-2">
        {filters.map((f) => {
          const active = filter === f.value;
          return (
            <button
              key={f.value}
              type="button"
              onClick={() => setFilter(f.value)}
              className="season-transition px-3 py-1.5 rounded-full text-xs"
              style={{
                backgroundColor: active
                  ? "color-mix(in srgb, var(--season-button) 22%, transparent)"
                  : "#161E19",
                color: active ? "#ECEFEA" : "#8B9A8E",
                border: active
                  ? "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)"
                  : "1px solid #1F2A24",
              }}
            >
              {f.label}
            </button>
          );
        })}
      </div>

      {query.isLoading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {query.isError && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(query.error)}
        </p>
      )}
      {!query.isLoading &&
        !query.isError &&
        trips.length === 0 &&
        (filter === "all" ? (
          <EmptyState onCreate={() => setOpenCreate(true)} />
        ) : (
          <FilteredEmpty filter={filter} />
        ))}

      {trips.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {trips.map((trip) => (
            <TripCard key={trip.id} trip={trip} />
          ))}
        </div>
      )}

      {openCreate && <CreateTripDialog onClose={() => setOpenCreate(false)} />}
      {openJoin && <JoinRoomDialog onClose={() => setOpenJoin(false)} />}
    </section>
  );
}

function TripCard({ trip }: { trip: Trip }) {
  const range = useMemo(
    () => formatRange(trip.start_date, trip.end_date),
    [trip.start_date, trip.end_date],
  );
  const isCompleted = trip.status === "completed";
  return (
    <Link
      href={`/trips/${trip.id}`}
      className="group block rounded-2xl overflow-hidden transition-transform hover:-translate-y-0.5"
      style={{
        backgroundColor: isCompleted ? "#0E1411" : "#161E19",
        border: `1px solid ${isCompleted ? "#161E19" : "#1F2A24"}`,
        opacity: isCompleted ? 0.65 : 1,
      }}
    >
      <div
        className="h-32 w-full"
        style={{
          background: trip.cover_image
            ? `url(${trip.cover_image}) center / cover`
            : "linear-gradient(135deg, #1F2A24 0%, #0B100D 100%)",
          filter: isCompleted ? "grayscale(0.6) brightness(0.7)" : undefined,
        }}
        aria-hidden
      />
      <div className="px-5 py-4 flex flex-col gap-2">
        <div className="flex items-start justify-between gap-3">
          <h3
            className="text-lg leading-snug line-clamp-2"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            {trip.title}
          </h3>
          <StatusBadge status={trip.status} />
        </div>
        {trip.description && (
          <p className="text-sm line-clamp-2" style={{ color: "#8B9A8E" }}>
            {trip.description}
          </p>
        )}
        <div
          className="flex items-center justify-between text-xs mt-1"
          style={{ color: "#6B7A6F" }}
        >
          <span>{range}</span>
          <span className="inline-flex items-center gap-1">
            <Users className="size-3.5" />
            {trip.member_count}
          </span>
        </div>
      </div>
    </Link>
  );
}

function StatusBadge({ status }: { status: TripStatus }) {
  const palette: Record<TripStatus, { bg: string; fg: string }> = {
    planning: { bg: "rgba(168, 224, 180, 0.14)", fg: "#A8E0B4" },
    ongoing: { bg: "rgba(232, 215, 184, 0.14)", fg: "#E8D7B8" },
    completed: { bg: "rgba(139, 154, 142, 0.18)", fg: "#8B9A8E" },
  };
  const colors = palette[status];
  return (
    <span
      className="text-[10px] tracking-wider uppercase px-2 py-0.5 rounded-full font-semibold shrink-0"
      style={{ backgroundColor: colors.bg, color: colors.fg }}
    >
      {status}
    </span>
  );
}

function FilteredEmpty({ filter }: { filter: Exclude<FilterValue, "all"> }) {
  const label = filter;
  return (
    <div
      className="rounded-2xl px-6 py-10 flex items-center justify-center text-center"
      style={{
        backgroundColor: "#121814",
        border: "1px dashed #1F2A24",
      }}
    >
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        No {label} trips
      </p>
    </div>
  );
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div
      className="rounded-2xl px-6 py-12 flex flex-col items-center text-center"
      style={{
        backgroundColor: "#121814",
        border: "1px dashed #1F2A24",
      }}
    >
      <h3
        className="text-xl"
        style={{
          fontFamily: "var(--font-display, Georgia, serif)",
          color: "#ECEFEA",
        }}
      >
        No trips yet
      </h3>
      <p className="text-sm mt-2 max-w-sm" style={{ color: "#8B9A8E" }}>
        Start by creating a trip, then invite your travel companions to plan
        together.
      </p>
      <button
        type="button"
        onClick={onCreate}
        className="season-transition mt-5 inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full"
        style={{
          backgroundColor: "var(--season-button)",
          color: "#0B100D",
        }}
      >
        <Plus className="size-4" />
        Create a trip
      </button>
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
