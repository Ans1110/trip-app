"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Bookmark,
  BookmarkCheck,
  CalendarPlus,
  Check,
  ChevronDown,
  Clock,
  Globe,
  Loader2,
  MapPin,
  Navigation,
  Phone,
} from "lucide-react";

import { ControlledSelect } from "@/components/ui/controlled-select";
import { errorMessage, usePlaceDetails } from "@/hooks/location-hooks";
import { useCreateItinerary, useTrips } from "@/hooks/trip-hooks";
import type { PlaceHours } from "@/lib/apis/location-api";
import type { Trip } from "@/lib/apis/trip-api";
import { addToTripSchema, type AddToTripFormInput } from "@/lib/schemas/location-schemas";
import { cn } from "@/lib/utils";

import { OpenBadge, PlacePhoto, RatingLine } from "./place-meta";

export type PlaceRef = {
  key: string;
  place_id?: string;
  name: string;
  address?: string;
  lat: number;
  lng: number;
  rating?: number;
  user_ratings_total?: number;
  types?: string[];
  price_level?: number;
  photo?: string;
  hours?: PlaceHours;
};

type Props = {
  place: PlaceRef;
  isSaved: boolean;
  saving: boolean;
  onToggleSave: () => void;
  onDirections: () => void;
};

export function PlaceDetailsPanel({ place, isSaved, saving, onToggleSave, onDirections }: Props) {
  const details = usePlaceDetails(place.place_id, undefined, { staleTime: 10 * 60_000 });
  const d = details.data;
  const photos = d?.photos?.length ? d.photos.slice(0, 8) : place.photo ? [place.photo] : [];
  const hours = d?.hours ?? place.hours;
  const [showWeek, setShowWeek] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  return (
    <article aria-label={place.name} className="flex flex-col">
      {photos.length > 0 ? (
        <div className="flex gap-1.5 overflow-x-auto snap-x snap-mandatory -mx-1 px-1 pb-1 hourly-scroll">
          {photos.map((name, i) => (
            <PlacePhoto
              key={name}
              name={name}
              width={i === 0 ? 800 : 480}
              alt={i === 0 ? `Photo of ${place.name}` : ""}
              seed={place.key}
              className={cn(
                "snap-start shrink-0 h-44 rounded-2xl",
                i === 0 ? "w-[78%]" : "w-[42%]",
              )}
            />
          ))}
        </div>
      ) : (
        <PlacePhoto alt="" seed={place.key} width={0} className="h-28 rounded-2xl" />
      )}

      <header className="pt-4 pb-3">
        <h2 className="text-[22px] leading-tight text-foreground">{place.name}</h2>
        <RatingLine
          className="mt-1.5"
          rating={d?.rating ?? place.rating}
          count={d?.user_ratings_total ?? place.user_ratings_total}
          priceLevel={d?.price_level ?? place.price_level}
          types={d?.types ?? place.types}
        />
        <OpenBadge hours={hours} className="mt-1" />
      </header>

      <div className="grid grid-cols-3 gap-2 pb-4 border-b border-border">
        <ActionButton primary icon={<Navigation className="size-[18px]" />} label="Directions" onClick={onDirections} />
        <ActionButton
          icon={
            saving ? (
              <Loader2 className="size-[18px] animate-spin" />
            ) : isSaved ? (
              <BookmarkCheck className="size-[18px]" />
            ) : (
              <Bookmark className="size-[18px]" />
            )
          }
          label={isSaved ? "Saved" : "Save"}
          pressed={isSaved}
          onClick={onToggleSave}
          disabled={saving}
        />
        <ActionButton
          icon={<CalendarPlus className="size-[18px]" />}
          label="Add to trip"
          pressed={addOpen}
          onClick={() => setAddOpen((v) => !v)}
        />
      </div>

      {addOpen && <AddToTripForm place={place} onDone={() => setAddOpen(false)} />}

      <ul className="flex flex-col py-2">
        {(d?.address ?? place.address) && (
          <InfoRow icon={<MapPin className="size-[18px]" />}>{d?.address ?? place.address}</InfoRow>
        )}
        {d?.opening_hours && d.opening_hours.length > 0 && (
          <InfoRow icon={<Clock className="size-[18px]" />}>
            <button
              type="button"
              aria-expanded={showWeek}
              onClick={() => setShowWeek((v) => !v)}
              className="inline-flex items-center gap-1 text-left text-foreground hover:underline underline-offset-2 outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {showWeek ? "Hide hours" : "Opening hours"}
              <ChevronDown className={cn("size-4 transition-transform", showWeek && "rotate-180")} />
            </button>
            {showWeek && (
              <ul className="mt-2 flex flex-col gap-0.5 text-[13px] text-muted-foreground">
                {d.opening_hours.map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            )}
          </InfoRow>
        )}
        {d?.phone_number && (
          <InfoRow icon={<Phone className="size-[18px]" />}>
            <a href={`tel:${d.phone_number.replace(/\s+/g, "")}`} className="text-sky-ink hover:underline underline-offset-2">
              {d.phone_number}
            </a>
          </InfoRow>
        )}
        {d?.website && (
          <InfoRow icon={<Globe className="size-[18px]" />}>
            <a
              href={d.website}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sky-ink hover:underline underline-offset-2 break-all"
            >
              {prettyUrl(d.website)}
            </a>
          </InfoRow>
        )}
      </ul>

      {details.isLoading && place.place_id && (
        <p className="text-sm text-muted-foreground py-2 flex items-center gap-2">
          <Loader2 className="size-4 animate-spin" /> Loading details…
        </p>
      )}
      {details.isError && (
        <p className="text-sm text-destructive py-2">{errorMessage(details.error)}</p>
      )}
    </article>
  );
}

function ActionButton({
  icon,
  label,
  onClick,
  primary,
  pressed,
  disabled,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  primary?: boolean;
  pressed?: boolean;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-pressed={primary ? undefined : pressed}
      className="group flex flex-col items-center gap-1.5 py-1 rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60"
    >
      <span
        className={cn(
          "grid place-items-center size-11 rounded-full transition-colors",
          primary
            ? "bg-primary text-primary-foreground shadow-[0_6px_16px_-6px_var(--season-button-shadow)] group-hover:brightness-[0.97]"
            : pressed
              ? "bg-pink-soft text-pink-ink"
              : "bg-sky-soft text-sky-ink group-hover:bg-muted",
        )}
      >
        {icon}
      </span>
      <span className={cn("text-xs", primary ? "text-foreground font-medium" : "text-muted-foreground")}>{label}</span>
    </button>
  );
}

function InfoRow({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <li className="flex gap-3.5 py-2.5 text-sm text-foreground">
      <span className="text-sky-ink mt-px shrink-0">{icon}</span>
      <div className="min-w-0">{children}</div>
    </li>
  );
}

function AddToTripForm({ place, onDone }: { place: PlaceRef; onDone: () => void }) {
  const trips = useTrips();
  const create = useCreateItinerary();
  const [added, setAdded] = useState<{ trip: Trip; day: number } | null>(null);
  const upcoming = useMemo(
    () => (trips.data ?? []).filter((t) => t.status !== "completed"),
    [trips.data],
  );
  const form = useForm<AddToTripFormInput>({
    resolver: zodResolver(addToTripSchema),
    defaultValues: { trip_id: "", day: 1 },
  });
  const tripId = form.watch("trip_id");
  const trip = upcoming.find((t) => t.id === tripId);
  const days = trip ? tripDayCount(trip) : 0;

  const submit = form.handleSubmit((v) => {
    const target = upcoming.find((t) => t.id === v.trip_id);
    if (!target) return;
    create.mutate(
      {
        id: v.trip_id,
        input: {
          day: v.day,
          title: place.name,
          location: place.address || place.name,
          latitude: place.lat,
          longitude: place.lng,
        },
      },
      { onSuccess: () => setAdded({ trip: target, day: v.day }) },
    );
  });

  if (added) {
    return (
      <div className="my-3 flex items-start gap-3 rounded-2xl bg-mint-soft px-4 py-3 text-sm">
        <Check className="size-[18px] text-mint-ink mt-px shrink-0" />
        <p className="text-foreground">
          Added to{" "}
          <Link href={`/trips/${added.trip.id}`} className="font-medium text-mint-ink hover:underline underline-offset-2">
            {added.trip.title}
          </Link>{" "}
          · Day {added.day}
          <button type="button" onClick={onDone} className="ml-2 text-muted-foreground hover:text-foreground">
            Done
          </button>
        </p>
      </div>
    );
  }

  if (trips.isLoading) {
    return <p className="my-3 text-sm text-muted-foreground">Loading your trips…</p>;
  }
  if (upcoming.length === 0) {
    return (
      <p className="my-3 rounded-2xl bg-muted px-4 py-3 text-sm text-muted-foreground">
        No upcoming trips yet.{" "}
        <Link href="/trips" className="text-mint-ink font-medium hover:underline underline-offset-2">
          Create one
        </Link>{" "}
        to add places to its itinerary.
      </p>
    );
  }

  return (
    <FormProvider {...form}>
      <form onSubmit={submit} className="my-3 flex flex-col gap-3 rounded-2xl bg-surface-sunken p-3">
        <div className="grid grid-cols-[1fr_7rem] gap-2">
          <ControlledSelect<AddToTripFormInput>
            name="trip_id"
            aria-label="Trip"
            placeholder="Choose a trip"
            options={upcoming.map((t) => ({ value: t.id, label: t.title }))}
          />
          <ControlledSelect<AddToTripFormInput>
            name="day"
            aria-label="Day"
            placeholder="Day"
            disabled={!trip}
            options={Array.from({ length: Math.max(days, 1) }, (_, i) => ({
              value: String(i + 1),
              label: `Day ${i + 1}`,
            }))}
          />
        </div>
        {create.isError && <p className="text-sm text-destructive">{errorMessage(create.error)}</p>}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onDone}
            className="h-9 px-4 rounded-full text-sm text-muted-foreground hover:bg-muted"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!trip || create.isPending}
            className="inline-flex items-center gap-1.5 h-9 px-4 rounded-full text-sm font-medium bg-primary text-primary-foreground disabled:opacity-60"
          >
            {create.isPending && <Loader2 className="size-4 animate-spin" />}
            Add to itinerary
          </button>
        </div>
      </form>
    </FormProvider>
  );
}

function tripDayCount(trip: Trip) {
  const start = Date.parse(trip.start_date);
  const end = Date.parse(trip.end_date);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return 1;
  return Math.min(60, Math.round((end - start) / 86_400_000) + 1);
}

function prettyUrl(url: string) {
  try {
    const u = new URL(url);
    return (u.hostname + (u.pathname === "/" ? "" : u.pathname)).replace(/^www\./, "");
  } catch {
    return url;
  }
}
