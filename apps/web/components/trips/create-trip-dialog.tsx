"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, MapPin, X } from "lucide-react";

import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { errorMessage, useCreateTrip } from "@/hooks/trip-hooks";
import {
  createTripSchema,
  type CreateTripFormInput,
} from "@/lib/schemas/trip-schemas";

import { Modal } from "./modal";
import { PlaceSearchPicker, type PickedPlace } from "./place-search-picker";

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
      location: "",
    },
  });
  const startDate = form.watch("start_date");
  const [picked, setPicked] = useState<PickedPlace | null>(null);

  const handlePick = (p: PickedPlace) => {
    setPicked(p);
    form.setValue("location", p.name, { shouldValidate: true });
  };
  const clearPick = () => {
    setPicked(null);
    form.setValue("location", "", { shouldValidate: true });
  };

  const onSubmit = (v: CreateTripFormInput) => {
    create.mutate(
      {
        title: v.title,
        description: v.description || undefined,
        start_date: v.start_date,
        end_date: v.end_date,
        location: v.location || undefined,
        latitude: picked?.lat,
        longitude: picked?.lng,
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
          <div className="grid grid-cols-2 gap-3">
            <ControlledInput<CreateTripFormInput>
              name="title"
              label="Title"
              placeholder="Japan · Spring 2026"
            />
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-[#8B9A8E]">
                Location (optional)
              </label>
              {picked ? (
                <div className="flex items-center gap-2 h-9 px-3 rounded-lg bg-[#0B100D] border border-[#1F2A24]">
                  <MapPin className="size-4 text-[#8B9A8E] shrink-0" />
                  <span className="text-sm text-[#ECEFEA] truncate flex-1">
                    {picked.name}
                  </span>
                  <button
                    type="button"
                    onClick={clearPick}
                    className="text-[#8B9A8E] hover:text-[#ECEFEA]"
                    aria-label="Clear location"
                  >
                    <X className="size-4" />
                  </button>
                </div>
              ) : (
                <PlaceSearchPicker
                  onPick={handlePick}
                  placeholder="Search a city or place"
                />
              )}
            </div>
          </div>

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
