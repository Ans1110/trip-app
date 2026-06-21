"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { ImagePlus, Loader2, Pencil, Trash2 } from "lucide-react";

import {
  errorMessage,
  useDeleteTrip,
  useUpdateTrip,
} from "@/hooks/trip-hooks";
import type { Trip, TripUpdatableStatus } from "@/lib/trip-api";

import { CoverImageDrawer } from "./cover-image-drawer";

const statusOptions: { value: TripUpdatableStatus; label: string }[] = [
  { value: "planning", label: "Planning" },
  { value: "ongoing", label: "Ongoing" },
];

export function TripOverview({ trip, canManage }: { trip: Trip; canManage: boolean }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [coverOpen, setCoverOpen] = useState(false);
  const [title, setTitle] = useState(trip.title);
  const [description, setDescription] = useState(trip.description);
  const [startDate, setStartDate] = useState(trip.start_date.slice(0, 10));
  const [endDate, setEndDate] = useState(trip.end_date.slice(0, 10));
  const [status, setStatus] = useState<TripUpdatableStatus>(
    trip.status === "completed" ? "ongoing" : trip.status,
  );

  const update = useUpdateTrip();
  const remove = useDeleteTrip();

  const handleSave = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    update.mutate(
      {
        id: trip.id,
        input: {
          title: title.trim(),
          description: description.trim(),
          start_date: startDate,
          end_date: endDate,
          status,
        },
      },
      { onSuccess: () => setEditing(false) },
    );
  };

  const handleDelete = () => {
    if (!window.confirm(`Delete "${trip.title}"? This cannot be undone.`)) return;
    remove.mutate(trip.id, {
      onSuccess: () => router.replace("/trips"),
    });
  };

  if (editing) {
    return (
      <form className="flex flex-col gap-4" onSubmit={handleSave}>
        <Field label="Title">
          <input
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className={inputClass}
            style={inputStyle}
          />
        </Field>
        <Field label="Description">
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={3}
            className={`${inputClass} resize-none`}
            style={inputStyle}
          />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Start date">
            <input
              required
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <Field label="End date">
            <input
              required
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              min={startDate || undefined}
              className={inputClass}
              style={inputStyle}
            />
          </Field>
        </div>
        <Field label="Status">
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as TripUpdatableStatus)}
            className={inputClass}
            style={inputStyle}
          >
            {statusOptions.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </Field>
        {update.isError && (
          <p className="text-sm" style={{ color: "#FCA5A5" }}>
            {errorMessage(update.error)}
          </p>
        )}
        <div className="flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => setEditing(false)}
            disabled={update.isPending}
            className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#8B9A8E" }}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={update.isPending}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {update.isPending && <Loader2 className="size-3.5 animate-spin" />}
            Save
          </button>
        </div>
      </form>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p
            className="text-[11px] font-medium tracking-[0.2em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            {trip.status}
          </p>
          <h2
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            {trip.title}
          </h2>
        </div>
        {canManage && (
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => setCoverOpen(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5"
              style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
            >
              <ImagePlus className="size-3.5" />
              Cover
            </button>
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5"
              style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
            >
              <Pencil className="size-3.5" />
              Edit
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={remove.isPending}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#FCA5A5" }}
            >
              {remove.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
              Delete
            </button>
          </div>
        )}
      </div>

      <Stat label="Dates" value={formatRange(trip.start_date, trip.end_date)} />
      <Stat
        label="Owner"
        value={trip.owner.name || trip.owner.email}
      />
      <Stat label="Members" value={String(trip.member_count)} />
      {trip.description && (
        <div>
          <p
            className="text-xs font-medium mb-2"
            style={{ color: "#8B9A8E" }}
          >
            Description
          </p>
          <p
            className="text-sm leading-relaxed whitespace-pre-wrap"
            style={{ color: "#ECEFEA" }}
          >
            {trip.description}
          </p>
        </div>
      )}
      {coverOpen && (
        <CoverImageDrawer trip={trip} onClose={() => setCoverOpen(false)} />
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium" style={{ color: "#8B9A8E" }}>
        {label}
      </p>
      <p className="text-sm mt-1" style={{ color: "#ECEFEA" }}>
        {value}
      </p>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-xs font-medium" style={{ color: "#8B9A8E" }}>
        {label}
      </span>
      {children}
    </label>
  );
}

const inputClass =
  "w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]";
const inputStyle: React.CSSProperties = {
  backgroundColor: "#161E19",
  border: "1px solid #1F2A24",
  color: "#ECEFEA",
};

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
