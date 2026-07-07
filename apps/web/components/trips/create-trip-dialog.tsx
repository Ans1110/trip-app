"use client";

import { useRouter } from "next/navigation";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { errorMessage, useCreateTrip } from "@/hooks/trip-hooks";
import {
  createTripSchema,
  type CreateTripFormInput,
} from "@/lib/schemas/trip-schemas";

import { Modal } from "./modal";

export function CreateTripDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const create = useCreateTrip();
  const form = useForm<CreateTripFormInput>({
    resolver: zodResolver(createTripSchema),
    defaultValues: {
      title: "",
      description: "",
      start_date: "",
      end_date: "",
    },
  });
  const startDate = form.watch("start_date");

  const onSubmit = (v: CreateTripFormInput) => {
    create.mutate(
      {
        title: v.title,
        description: v.description || undefined,
        start_date: v.start_date,
        end_date: v.end_date,
      },
      {
        onSuccess: (trip) => {
          onClose();
          router.push(`/trips/${trip.id}`);
        },
      },
    );
  };

  const submitError = errorMessage(create.error);

  return (
    <Modal title="Create a trip" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<CreateTripFormInput>
            name="title"
            label="Title"
            placeholder="Japan · Spring 2026"
          />

          <ControlledTextarea<CreateTripFormInput>
            name="description"
            label="Description (optional)"
            placeholder="What's this trip about?"
            rows={3}
          />

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<CreateTripFormInput>
              name="start_date"
              label="Start date"
            />
            <ControlledDatePicker<CreateTripFormInput>
              name="end_date"
              label="End date"
              min={startDate || undefined}
            />
          </div>

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
