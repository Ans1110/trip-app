"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ImagePlus, Loader2, Pencil, Trash2 } from "lucide-react";

import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import {
  errorMessage,
  useDeleteTrip,
  useUpdateTrip,
} from "@/hooks/trip-hooks";
import type { Trip, TripUpdatableStatus } from "@/lib/trip-api";
import {
  updateTripSchema,
  type UpdateTripFormInput,
} from "@/lib/trip-schemas";

import { CoverImageDrawer } from "./cover-image-drawer";

const statusOptions = [
  { value: "planning", label: "Planning" },
  { value: "ongoing", label: "Ongoing" },
];

export function TripOverview({ trip, canManage }: { trip: Trip; canManage: boolean }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [coverOpen, setCoverOpen] = useState(false);

  const defaults: UpdateTripFormInput = {
    title: trip.title,
    description: trip.description,
    start_date: trip.start_date.slice(0, 10),
    end_date: trip.end_date.slice(0, 10),
    status: (trip.status === "completed" ? "ongoing" : trip.status) as TripUpdatableStatus,
  };
  const form = useForm<UpdateTripFormInput>({
    resolver: zodResolver(updateTripSchema),
    defaultValues: defaults,
  });
  const startDate = form.watch("start_date");

  const update = useUpdateTrip();
  const remove = useDeleteTrip();

  const handleSave = (v: UpdateTripFormInput) => {
    update.mutate(
      {
        id: trip.id,
        input: {
          title: v.title.trim(),
          description: v.description.trim(),
          start_date: v.start_date,
          end_date: v.end_date,
          status: v.status,
        },
      },
      { onSuccess: () => setEditing(false) },
    );
  };

  const handleCancel = () => {
    form.reset(defaults);
    setEditing(false);
  };

  const handleDelete = () => {
    if (!window.confirm(`Delete "${trip.title}"? This cannot be undone.`)) return;
    remove.mutate(trip.id, {
      onSuccess: () => router.replace("/trips"),
    });
  };

  if (editing) {
    return (
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(handleSave)}
          noValidate
        >
          <ControlledInput<UpdateTripFormInput> name="title" label="Title" />
          <ControlledTextarea<UpdateTripFormInput>
            name="description"
            label="Description"
            rows={3}
          />
          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<UpdateTripFormInput>
              name="start_date"
              label="Start date"
            />
            <ControlledDatePicker<UpdateTripFormInput>
              name="end_date"
              label="End date"
              min={startDate || undefined}
            />
          </div>
          <ControlledSelect<UpdateTripFormInput>
            name="status"
            label="Status"
            options={statusOptions}
          />
          {update.isError && (
            <p className="text-sm text-[#FCA5A5]">
              {errorMessage(update.error)}
            </p>
          )}
          <div className="flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={handleCancel}
              disabled={update.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={update.isPending}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {update.isPending && <Loader2 className="size-3.5 animate-spin" />}
              Save
            </button>
          </div>
        </form>
      </FormProvider>
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
            className="text-3xl tracking-tight text-[#ECEFEA]"
            style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
          >
            {trip.title}
          </h2>
        </div>
        {canManage && (
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => setCoverOpen(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 text-[#ECEFEA] border border-[#1F2A24]"
            >
              <ImagePlus className="size-3.5" />
              Cover
            </button>
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 text-[#ECEFEA] border border-[#1F2A24]"
            >
              <Pencil className="size-3.5" />
              Edit
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={remove.isPending}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60 text-[#FCA5A5]"
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
      <Stat label="Owner" value={trip.owner.name || trip.owner.email} />
      <Stat label="Members" value={String(trip.member_count)} />
      {trip.description && (
        <div>
          <p className="text-xs font-medium mb-2 text-[#8B9A8E]">Description</p>
          <p className="text-sm leading-relaxed whitespace-pre-wrap text-[#ECEFEA]">
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
      <p className="text-xs font-medium text-[#8B9A8E]">{label}</p>
      <p className="text-sm mt-1 text-[#ECEFEA]">{value}</p>
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
