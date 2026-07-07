"use client";

import { useMemo } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, Trash2 } from "lucide-react";

import { ControlledCheckbox } from "@/components/ui/controlled-checkbox";
import { ControlledColorPalette } from "@/components/ui/controlled-color-palette";
import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { ControlledTimePicker } from "@/components/ui/controlled-time-picker";
import { Modal } from "@/components/trips/modal";
import {
  errorMessage,
  useDeleteEvent,
  useUpdateEvent,
} from "@/hooks/calendar-hooks";
import type { CalendarEvent } from "@/lib/apis/calendar-api";
import {
  updateEventSchema,
  type UpdateEventFormInput,
} from "@/lib/schemas/calendar-schemas";

const VISIBILITY_OPTIONS = [
  { value: "private", label: "Private" },
  { value: "friends", label: "Friends" },
];

export function EditEventDialog({
  event,
  onClose,
}: {
  event: CalendarEvent;
  onClose: () => void;
}) {
  const update = useUpdateEvent();
  const remove = useDeleteEvent();

  const defaults = useMemo<UpdateEventFormInput>(() => {
    const s = splitISO(event.start_at);
    const e = splitISO(event.end_at);
    return {
      title: event.title,
      description: event.description ?? "",
      location: event.location ?? "",
      start_date: s.date,
      start_time: s.time,
      end_date: e.date,
      end_time: e.time,
      all_day: event.all_day,
      color: event.color ?? "",
      visibility: event.visibility === "private" ? "private" : "friends",
    };
  }, [event]);

  const form = useForm<UpdateEventFormInput>({
    resolver: zodResolver(updateEventSchema),
    defaultValues: defaults,
  });
  const startDate = form.watch("start_date");
  const startTime = form.watch("start_time");
  const endDate = form.watch("end_date");
  const allDay = form.watch("all_day");

  const onSubmit = (v: UpdateEventFormInput) => {
    const startISO = combineToISO(
      v.start_date,
      v.all_day ? "00:00" : v.start_time,
    );
    const endISO = combineToISO(v.end_date, v.all_day ? "23:59" : v.end_time);
    update.mutate(
      {
        id: event.id,
        input: {
          title: v.title,
          description: v.description || undefined,
          location: v.location || undefined,
          start_at: startISO,
          end_at: endISO,
          time_zone: resolvedTimeZone(),
          all_day: v.all_day,
          color: v.color || undefined,
          visibility: v.visibility,
          version: event.version,
        },
      },
      { onSuccess: () => onClose() },
    );
  };

  const onDelete = () => {
    if (!confirm("Delete this event?")) return;
    remove.mutate(event.id, { onSuccess: () => onClose() });
  };

  const submitError = errorMessage(update.error) ?? errorMessage(remove.error);
  const pending = update.isPending || remove.isPending;

  return (
    <Modal title="Edit event" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<UpdateEventFormInput> name="title" label="Title" />

          <ControlledTextarea<UpdateEventFormInput>
            name="description"
            label="Description"
            rows={3}
          />

          <ControlledInput<UpdateEventFormInput>
            name="location"
            label="Location"
          />

          <ControlledCheckbox<UpdateEventFormInput>
            name="all_day"
            label="All day"
          />

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<UpdateEventFormInput>
              name="start_date"
              label="Starts"
            />
            <ControlledTimePicker<UpdateEventFormInput>
              name="start_time"
              label="Time"
              disabled={allDay}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<UpdateEventFormInput>
              name="end_date"
              label="Ends"
              min={startDate || undefined}
            />
            <ControlledTimePicker<UpdateEventFormInput>
              name="end_time"
              label="Time"
              disabled={allDay}
              hint={
                !allDay && startDate === endDate && startTime
                  ? `On or after ${startTime}`
                  : undefined
              }
            />
          </div>

          <ControlledColorPalette<UpdateEventFormInput>
            name="color"
            label="Color"
          />

          <ControlledSelect<UpdateEventFormInput>
            name="visibility"
            label="Visibility"
            options={VISIBILITY_OPTIONS}
          />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-between pt-2">
            <button
              type="button"
              onClick={onDelete}
              disabled={pending}
              className="inline-flex items-center gap-1.5 px-3 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-40"
              style={{ color: "#FCA5A5" }}
            >
              <Trash2 className="size-3.5" />
              Delete
            </button>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onClose}
                disabled={pending}
                className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={pending}
                className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
                style={{ backgroundColor: "var(--season-button)" }}
              >
                {update.isPending && (
                  <Loader2 className="size-3.5 animate-spin" />
                )}
                Save
              </button>
            </div>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}

function pad(n: number): string {
  return n.toString().padStart(2, "0");
}

function splitISO(iso: string): { date: string; time: string } {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { date: "", time: "" };
  return {
    date: `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
    time: `${pad(d.getHours())}:${pad(d.getMinutes())}`,
  };
}

function combineToISO(date: string, time: string): string {
  const local = `${date}T${time}`;
  const d = new Date(local);
  if (Number.isNaN(d.getTime())) return local;
  return d.toISOString();
}

function resolvedTimeZone(): string | undefined {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || undefined;
  } catch {
    return undefined;
  }
}
