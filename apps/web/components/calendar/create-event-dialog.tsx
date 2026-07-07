"use client";

import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { ControlledCheckbox } from "@/components/ui/controlled-checkbox";
import { ControlledColorPalette } from "@/components/ui/controlled-color-palette";
import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { ControlledTimePicker } from "@/components/ui/controlled-time-picker";
import { Modal } from "@/components/trips/modal";
import { errorMessage, useCreateEvent } from "@/hooks/calendar-hooks";
import {
  createEventSchema,
  type CreateEventFormInput,
} from "@/lib/schemas/calendar-schemas";

const VISIBILITY_OPTIONS = [
  { value: "private", label: "Private" },
  { value: "friends", label: "Friends" },
];

export function CreateEventDialog({
  onClose,
  defaultDate,
}: {
  onClose: () => void;
  defaultDate?: string;
}) {
  const create = useCreateEvent();
  const start = defaultDate ?? todayDate();
  const form = useForm<CreateEventFormInput>({
    resolver: zodResolver(createEventSchema),
    defaultValues: {
      title: "",
      description: "",
      location: "",
      start_date: start,
      start_time: "09:00",
      end_date: start,
      end_time: "10:00",
      all_day: false,
      color: "",
      visibility: "private",
    },
  });
  const startDate = form.watch("start_date");
  const startTime = form.watch("start_time");
  const endDate = form.watch("end_date");
  const allDay = form.watch("all_day");

  const onSubmit = (v: CreateEventFormInput) => {
    const startISO = combineToISO(
      v.start_date,
      v.all_day ? "00:00" : v.start_time,
    );
    const endISO = combineToISO(v.end_date, v.all_day ? "23:59" : v.end_time);
    create.mutate(
      {
        title: v.title,
        description: v.description || undefined,
        location: v.location || undefined,
        start_at: startISO,
        end_at: endISO,
        time_zone: resolvedTimeZone(),
        all_day: v.all_day,
        color: v.color || undefined,
        visibility: v.visibility,
      },
      { onSuccess: () => onClose() },
    );
  };

  const submitError = errorMessage(create.error);

  return (
    <Modal title="New event" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<CreateEventFormInput>
            name="title"
            label="Title"
            placeholder="Flight to Tokyo"
          />

          <ControlledTextarea<CreateEventFormInput>
            name="description"
            label="Description (optional)"
            placeholder="Any details worth remembering?"
            rows={3}
          />

          <ControlledInput<CreateEventFormInput>
            name="location"
            label="Location (optional)"
            placeholder="Haneda Airport"
          />

          <ControlledCheckbox<CreateEventFormInput>
            name="all_day"
            label="All day"
          />

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<CreateEventFormInput>
              name="start_date"
              label="Starts"
            />
            <ControlledTimePicker<CreateEventFormInput>
              name="start_time"
              label="Time"
              disabled={allDay}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<CreateEventFormInput>
              name="end_date"
              label="Ends"
              min={startDate || undefined}
            />
            <ControlledTimePicker<CreateEventFormInput>
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

          <ControlledColorPalette<CreateEventFormInput>
            name="color"
            label="Color"
          />

          <ControlledSelect<CreateEventFormInput>
            name="visibility"
            label="Visibility"
            options={VISIBILITY_OPTIONS}
          />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={create.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={create.isPending}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {create.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              Create
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}

function pad(n: number): string {
  return n.toString().padStart(2, "0");
}

function todayDate(): string {
  const d = new Date();
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
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
