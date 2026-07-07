"use client";

import { useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { GripVertical, Loader2, MapPin, Plus, Trash2 } from "lucide-react";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import {
  errorMessage,
  useCreateItinerary,
  useDeleteItinerary,
  useItinerary,
  useReorderItinerary,
  useUpdateItinerary,
} from "@/hooks/trip-hooks";
import type { Itinerary, Trip } from "@/lib/apis/trip-api";
import {
  createItinerarySchema,
  updateItinerarySchema,
  type CreateItineraryFormInput,
  type UpdateItineraryFormInput,
} from "@/lib/schemas/trip-schemas";

const ItineraryMap = dynamic(() => import("./itinerary-map"), { ssr: false });

export function TripItinerary({ trip }: { trip: Trip }) {
  const tripId = trip.id;
  const query = useItinerary(tripId);

  const grouped = useMemo(() => {
    const items = query.data ?? [];
    const byDay = new Map<number, Itinerary[]>();
    for (const item of items) {
      const list = byDay.get(item.day) ?? [];
      list.push(item);
      byDay.set(item.day, list);
    }
    for (const list of byDay.values()) {
      list.sort((a, b) => a.sort_order - b.sort_order);
    }
    return [...byDay.entries()].sort(([a], [b]) => a - b);
  }, [query.data]);

  const mapMarkers = useMemo(() => {
    const items = query.data ?? [];
    return items
      .filter(
        (i): i is Itinerary & { latitude: number; longitude: number } =>
          typeof i.latitude === "number" && typeof i.longitude === "number",
      )
      .map((i) => ({
        id: i.id,
        lat: i.latitude,
        lng: i.longitude,
        label: i.title || i.location || "Stop",
      }));
  }, [query.data]);

  const itemCount = query.data?.length ?? 0;

  return (
    <section className="flex flex-col gap-6">
      <NewItemForm tripId={tripId} />

      {mapMarkers.length > 0 && <ItineraryMap markers={mapMarkers} />}

      {query.isLoading && <p className="text-sm text-[#8B9A8E]">Loading…</p>}
      {query.isError && (
        <p className="text-sm text-[#FCA5A5]">{errorMessage(query.error)}</p>
      )}
      {!query.isLoading && !query.isError && itemCount === 0 && (
        <p className="text-sm text-[#8B9A8E]">
          No itinerary items yet. Add the first stop above.
        </p>
      )}

      {grouped.map(([day, list]) => (
        <DayGroup key={day} tripId={tripId} day={day} items={list} />
      ))}
    </section>
  );
}

function DayGroup({
  tripId,
  day,
  items,
}: {
  tripId: string;
  day: number;
  items: Itinerary[];
}) {
  const reorder = useReorderItinerary();
  const [orderOverride, setOrderOverride] = useState<Itinerary[] | null>(null);
  const display = orderOverride ?? items;

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleDragEnd = (e: DragEndEvent) => {
    const { active, over } = e;
    if (!over || active.id === over.id) return;
    const oldIndex = display.findIndex((i) => i.id === active.id);
    const newIndex = display.findIndex((i) => i.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const next = arrayMove(display, oldIndex, newIndex);
    setOrderOverride(next);
    reorder.mutate(
      { id: tripId, input: { item_ids: next.map((i) => i.id) } },
      { onSettled: () => setOrderOverride(null) },
    );
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3
          className="text-sm font-medium tracking-wider uppercase"
          style={{ color: "var(--season-button)" }}
        >
          Day {day}
        </h3>
        {reorder.isPending && (
          <Loader2 className="size-3.5 animate-spin text-[#8B9A8E]" />
        )}
      </div>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={display.map((i) => i.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="flex flex-col gap-2">
            {display.map((item) => (
              <SortableItemRow key={item.id} tripId={tripId} item={item} />
            ))}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}

function NewItemForm({ tripId }: { tripId: string }) {
  const create = useCreateItinerary();
  const form = useForm<CreateItineraryFormInput>({
    resolver: zodResolver(createItinerarySchema),
    defaultValues: { day: "1", title: "", location: "" },
  });

  const onSubmit = (v: CreateItineraryFormInput) => {
    create.mutate(
      {
        id: tripId,
        input: {
          day: Number(v.day),
          title: v.title || undefined,
          location: v.location || undefined,
        },
      },
      {
        onSuccess: () => form.reset({ day: v.day, title: "", location: "" }),
      },
    );
  };

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        noValidate
        className="flex flex-col gap-3 p-4 rounded-xl bg-[#121814] border border-[#1F2A24]"
      >
        <div className="grid grid-cols-1 sm:grid-cols-[120px_1fr_1fr_auto] gap-2 items-end">
          <ControlledInput<CreateItineraryFormInput>
            name="day"
            label="Day"
            inputMode="numeric"
            placeholder="1"
          />
          <ControlledInput<CreateItineraryFormInput>
            name="title"
            label="Title"
            placeholder="Senso-ji visit"
          />
          <ControlledInput<CreateItineraryFormInput>
            name="location"
            label="Location"
            placeholder="Tokyo"
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
            style={{ backgroundColor: "var(--season-button)" }}
          >
            {create.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Plus className="size-4" />
            )}
            Add
          </button>
        </div>
        {create.isError && (
          <p className="text-xs text-[#FCA5A5]">{errorMessage(create.error)}</p>
        )}
      </form>
    </FormProvider>
  );
}

function SortableItemRow({
  tripId,
  item,
}: {
  tripId: string;
  item: Itinerary;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  };
  return (
    <div ref={setNodeRef} style={style}>
      <ItemRow
        tripId={tripId}
        item={item}
        dragHandle={
          <button
            type="button"
            aria-label="Reorder"
            className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 cursor-grab active:cursor-grabbing touch-none text-[#8B9A8E]"
            {...attributes}
            {...listeners}
          >
            <GripVertical className="size-3.5" />
          </button>
        }
      />
    </div>
  );
}

function ItemRow({
  tripId,
  item,
  dragHandle,
}: {
  tripId: string;
  item: Itinerary;
  dragHandle?: React.ReactNode;
}) {
  const [editing, setEditing] = useState(false);
  const update = useUpdateItinerary();
  const remove = useDeleteItinerary();
  const defaults: UpdateItineraryFormInput = {
    title: item.title,
    location: item.location,
    description: item.description,
    latitude: item.latitude !== undefined ? String(item.latitude) : "",
    longitude: item.longitude !== undefined ? String(item.longitude) : "",
  };
  const form = useForm<UpdateItineraryFormInput>({
    resolver: zodResolver(updateItinerarySchema),
    defaultValues: defaults,
  });

  const onSave = (v: UpdateItineraryFormInput) => {
    update.mutate(
      {
        id: tripId,
        itemId: item.id,
        input: {
          title: v.title,
          description: v.description,
          location: v.location,
          latitude: v.latitude === "" ? null : Number(v.latitude),
          longitude: v.longitude === "" ? null : Number(v.longitude),
        },
      },
      { onSuccess: () => setEditing(false) },
    );
  };

  const handleCancel = () => {
    form.reset(defaults);
    setEditing(false);
  };

  return (
    <div className="flex items-start gap-3 px-4 py-3 rounded-lg bg-[#121814] border border-[#1F2A24]">
      {dragHandle}
      <FormProvider {...form}>
        <form
          onSubmit={form.handleSubmit(onSave)}
          noValidate
          className="flex-1 min-w-0 flex flex-col gap-1.5"
        >
          {editing ? (
            <div className="flex flex-col gap-2">
              <ControlledInput<UpdateItineraryFormInput>
                name="title"
                placeholder="Title"
              />
              <ControlledInput<UpdateItineraryFormInput>
                name="location"
                placeholder="Location"
              />
              <div className="grid grid-cols-2 gap-2">
                <ControlledInput<UpdateItineraryFormInput>
                  name="latitude"
                  placeholder="Latitude"
                  inputMode="decimal"
                />
                <ControlledInput<UpdateItineraryFormInput>
                  name="longitude"
                  placeholder="Longitude"
                  inputMode="decimal"
                />
              </div>
              <ControlledTextarea<UpdateItineraryFormInput>
                name="description"
                placeholder="Description"
                rows={2}
              />
            </div>
          ) : (
            <>
              <p className="text-sm text-[#ECEFEA]">
                {item.title || "Untitled"}
              </p>
              {(item.location ||
                (item.latitude !== undefined &&
                  item.longitude !== undefined)) && (
                <div className="flex items-center gap-1 text-[#8B9A8E]">
                  <MapPin className="size-3" />
                  <span className="text-xs">
                    {item.location || ""}
                    {item.location &&
                    item.latitude !== undefined &&
                    item.longitude !== undefined
                      ? " · "
                      : ""}
                    {item.latitude !== undefined && item.longitude !== undefined
                      ? `${item.latitude.toFixed(4)}, ${item.longitude.toFixed(4)}`
                      : ""}
                  </span>
                </div>
              )}
              {item.description && (
                <p className="text-xs whitespace-pre-wrap text-[#6B7A6F]">
                  {item.description}
                </p>
              )}
            </>
          )}
        </form>
      </FormProvider>
      <div className="flex items-center gap-1.5 shrink-0">
        {editing ? (
          <>
            <button
              type="button"
              onClick={handleCancel}
              className="text-xs px-2 py-1 rounded-full hover:bg-white/5 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={form.handleSubmit(onSave)}
              disabled={update.isPending}
              className="season-transition inline-flex items-center gap-1 text-xs px-3 py-1 rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {update.isPending && <Loader2 className="size-3 animate-spin" />}
              Save
            </button>
          </>
        ) : (
          <>
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="text-xs px-2 py-1 rounded-full hover:bg-white/5 text-[#8B9A8E]"
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => remove.mutate({ id: tripId, itemId: item.id })}
              disabled={remove.isPending}
              aria-label="Delete"
              className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60 text-[#FCA5A5]"
            >
              {remove.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
            </button>
          </>
        )}
      </div>
    </div>
  );
}
