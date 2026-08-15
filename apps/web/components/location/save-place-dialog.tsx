"use client";

import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { Modal } from "@/components/trips/modal";
import {
  errorMessage,
  useCreateSavedPlace,
  useUpdateSavedPlace,
} from "@/hooks/location-hooks";
import type { SavedPlace } from "@/lib/apis/location-api";
import {
  savedPlaceSchema,
  type SavedPlaceFormInput,
} from "@/lib/schemas/location-schemas";

const CATEGORY_NONE = "__none";
const CATEGORY_OPTIONS = [
  { value: CATEGORY_NONE, label: "Uncategorized" },
  { value: "food", label: "Food" },
  { value: "lodging", label: "Lodging" },
  { value: "sight", label: "Sight" },
  { value: "activity", label: "Activity" },
  { value: "shopping", label: "Shopping" },
  { value: "transport", label: "Transport" },
];

const toFormCategory = (c: string | undefined | null) =>
  c && c.length > 0 ? c : CATEGORY_NONE;
const fromFormCategory = (c: string) =>
  c === CATEGORY_NONE ? undefined : c || undefined;

type Draft = {
  name?: string;
  address?: string;
  lat: number;
  lng: number;
  place_id?: string;
  trip_id?: string;
};

type Props =
  | { mode: "create"; draft: Draft; onClose: () => void }
  | { mode: "edit"; place: SavedPlace; onClose: () => void };

export function SavePlaceDialog(props: Props) {
  const create = useCreateSavedPlace();
  const update = useUpdateSavedPlace();
  const isEdit = props.mode === "edit";

  const defaults: SavedPlaceFormInput = isEdit
    ? {
        name: props.place.name,
        address: props.place.address ?? "",
        lat: props.place.lat,
        lng: props.place.lng,
        place_id: props.place.place_id ?? "",
        category: toFormCategory(props.place.category),
        notes: props.place.notes ?? "",
        color: props.place.color ?? "",
        trip_id: props.place.trip_id ?? "",
      }
    : {
        name: props.draft.name ?? "",
        address: props.draft.address ?? "",
        lat: props.draft.lat,
        lng: props.draft.lng,
        place_id: props.draft.place_id ?? "",
        category: CATEGORY_NONE,
        notes: "",
        color: "",
        trip_id: props.draft.trip_id ?? "",
      };

  const form = useForm<SavedPlaceFormInput>({
    resolver: zodResolver(savedPlaceSchema),
    defaultValues: defaults,
  });

  const pending = create.isPending || update.isPending;
  const submitError = errorMessage(
    isEdit ? update.error : create.error,
  );

  const onSubmit = (v: SavedPlaceFormInput) => {
    const category = fromFormCategory(v.category);
    if (isEdit) {
      update.mutate(
        {
          id: props.place.id,
          input: {
            name: v.name,
            address: v.address || undefined,
            lat: v.lat,
            lng: v.lng,
            place_id: v.place_id || undefined,
            category,
            notes: v.notes || undefined,
            color: v.color || undefined,
            trip_id: v.trip_id || undefined,
            unset_trip_id:
              !v.trip_id && !!props.place.trip_id ? true : undefined,
          },
        },
        { onSuccess: () => props.onClose() },
      );
    } else {
      create.mutate(
        {
          name: v.name,
          address: v.address || undefined,
          lat: v.lat,
          lng: v.lng,
          place_id: v.place_id || undefined,
          category,
          notes: v.notes || undefined,
          color: v.color || undefined,
          trip_id: v.trip_id || undefined,
        },
        { onSuccess: () => props.onClose() },
      );
    }
  };

  return (
    <Modal
      title={isEdit ? "Edit saved place" : "Save place"}
      onClose={props.onClose}
      size="lg"
    >
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<SavedPlaceFormInput>
            name="name"
            label="Name"
            placeholder="Senso-ji Temple"
          />
          <ControlledInput<SavedPlaceFormInput>
            name="address"
            label="Address"
            placeholder="Optional"
          />
          <div className="grid grid-cols-2 gap-3">
            <ControlledInput<SavedPlaceFormInput>
              name="lat"
              label="Latitude"
              inputMode="decimal"
              type="number"
              step="any"
              valueAsNumber
            />
            <ControlledInput<SavedPlaceFormInput>
              name="lng"
              label="Longitude"
              inputMode="decimal"
              type="number"
              step="any"
              valueAsNumber
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <ControlledSelect<SavedPlaceFormInput>
              name="category"
              label="Category"
              options={CATEGORY_OPTIONS}
            />
            <ControlledInput<SavedPlaceFormInput>
              name="color"
              label="Color tag"
              placeholder="#B5D086 or name"
            />
          </div>
          <ControlledInput<SavedPlaceFormInput>
            name="trip_id"
            label="Trip ID (optional)"
            placeholder="Bind to a trip"
          />
          <ControlledTextarea<SavedPlaceFormInput>
            name="notes"
            label="Notes"
            placeholder="Anything to remember about this place"
            rows={3}
          />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={props.onClose}
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
              {pending && <Loader2 className="size-3.5 animate-spin" />}
              {isEdit ? "Save" : "Create"}
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}
