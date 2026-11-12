"use client";

import { useEffect, useId, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowUpDown, Bike, Car, CircleDot, Footprints, LocateFixed, Loader2, MapPin, TrainFront } from "lucide-react";

import type { Route, TravelMode } from "@/lib/apis/location-api";
import { searchSchema, type SearchFormInput } from "@/lib/schemas/location-schemas";
import { cn } from "@/lib/utils";

import { usePlaceSuggestions, type SearchBias } from "./place-search-box";

export type Endpoint = { label: string; lat: number; lng: number; isMe?: boolean };

const MODES: { mode: TravelMode; label: string; icon: React.ReactNode }[] = [
  { mode: "driving", label: "Drive", icon: <Car className="size-[18px]" /> },
  { mode: "transit", label: "Transit", icon: <TrainFront className="size-[18px]" /> },
  { mode: "walking", label: "Walk", icon: <Footprints className="size-[18px]" /> },
  { mode: "bicycling", label: "Bike", icon: <Bike className="size-[18px]" /> },
];

type Props = {
  from: Endpoint | null;
  to: Endpoint | null;
  mode: TravelMode;
  bias: SearchBias | null;
  route: Route | null;
  pending: boolean;
  error: string | null;
  locating: boolean;
  onFrom: (e: Endpoint | null) => void;
  onTo: (e: Endpoint | null) => void;
  onSwap: () => void;
  onMode: (m: TravelMode) => void;
  onUseMyLocation: () => void;
};

export function DirectionsPanel(props: Props) {
  const { from, to, mode, route, pending, error, onSwap, onMode } = props;
  const steps = route?.legs?.flatMap((l) => l.steps ?? []) ?? [];

  return (
    <section aria-label="Directions" className="flex flex-col gap-4">
      <div className="flex items-start gap-2">
        <div className="flex-1 min-w-0 flex flex-col gap-2">
          <EndpointField
            icon={<CircleDot className="size-4 text-mint-ink" />}
            connector
            label="Starting point"
            placeholder="Choose starting point"
            value={from}
            bias={props.bias}
            allowMe
            locating={props.locating}
            onUseMyLocation={props.onUseMyLocation}
            onChange={props.onFrom}
          />
          <EndpointField
            icon={<MapPin className="size-4 text-pink-ink" />}
            label="Destination"
            placeholder="Choose destination"
            value={to}
            bias={props.bias}
            onChange={props.onTo}
          />
        </div>
        <button
          type="button"
          onClick={onSwap}
          disabled={!from && !to}
          aria-label="Swap start and destination"
          className="mt-[26px] grid place-items-center size-9 rounded-full text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-40 shrink-0 outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ArrowUpDown className="size-[18px]" />
        </button>
      </div>

      <div role="radiogroup" aria-label="Travel mode" className="grid grid-cols-4 gap-1 p-1 rounded-full bg-surface-sunken">
        {MODES.map((m) => {
          const active = m.mode === mode;
          return (
            <button
              key={m.mode}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => onMode(m.mode)}
              className={cn(
                "flex items-center justify-center gap-1.5 h-9 rounded-full text-[13px] transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
                active ? "bg-card text-mint-ink font-medium shadow-soft" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {m.icon}
              <span className="hidden sm:inline">{m.label}</span>
            </button>
          );
        })}
      </div>

      {!from || !to ? (
        <p className="text-sm text-muted-foreground px-1">
          {!from ? "Choose a starting point, or use your location." : "Choose a destination."}
        </p>
      ) : pending ? (
        <p className="flex items-center gap-2 text-sm text-muted-foreground px-1">
          <Loader2 className="size-4 animate-spin" /> Finding the best route…
        </p>
      ) : error ? (
        <p className="rounded-2xl bg-pink-soft px-4 py-3 text-sm text-destructive">{error}</p>
      ) : route ? (
        <>
          <div className="rounded-2xl bg-mint-soft px-4 py-3">
            <p className="flex items-baseline gap-2">
              <span className="text-2xl font-display font-bold text-foreground">
                {formatDuration(route.duration_seconds)}
              </span>
              <span className="text-sm text-muted-foreground">{formatDistance(route.distance_meters)}</span>
            </p>
            {route.summary && <p className="text-sm text-muted-foreground mt-0.5">via {route.summary}</p>}
          </div>
          {steps.length > 0 && (
            <ol className="flex flex-col">
              {steps.map((s, i) => (
                <li key={i} className="flex gap-3 py-2.5 border-b border-border last:border-0">
                  <span className="grid place-items-center size-6 rounded-full bg-sky-soft text-sky-ink text-xs font-medium tabular-nums shrink-0">
                    {i + 1}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm text-foreground">{s.instruction || "Continue"}</p>
                    {s.distance_meters > 0 && (
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {formatDistance(s.distance_meters)}
                        {s.duration_seconds > 0 && ` · ${formatDuration(s.duration_seconds)}`}
                      </p>
                    )}
                  </div>
                </li>
              ))}
            </ol>
          )}
        </>
      ) : null}
    </section>
  );
}

function EndpointField({
  icon,
  connector,
  label,
  placeholder,
  value,
  bias,
  allowMe,
  locating,
  onUseMyLocation,
  onChange,
}: {
  icon: React.ReactNode;
  connector?: boolean;
  label: string;
  placeholder: string;
  value: Endpoint | null;
  bias: SearchBias | null;
  allowMe?: boolean;
  locating?: boolean;
  onUseMyLocation?: () => void;
  onChange: (e: Endpoint | null) => void;
}) {
  const form = useForm<SearchFormInput>({
    resolver: zodResolver(searchSchema),
    defaultValues: { q: value?.label ?? "" },
  });
  const { register, setValue, watch } = form;
  const q = watch("q") ?? "";
  const [open, setOpen] = useState(false);
  const listId = useId();
  const editing = open && q !== (value?.label ?? "");
  const suggestions = usePlaceSuggestions(q, bias, editing);

  useEffect(() => {
    setValue("q", value?.label ?? "");
  }, [value?.label, setValue]);

  const choose = (e: Endpoint) => {
    setOpen(false);
    setValue("q", e.label);
    onChange(e);
  };

  const showList = open && ((allowMe && !value?.isMe) || suggestions.items.length > 0);

  return (
    <div className="grid grid-cols-[16px_1fr] gap-x-2">
      <span className="relative grid place-items-center h-10" aria-hidden>
        {icon}
        {connector && !showList && (
          <span className="absolute top-[30px] h-[28px] border-l-2 border-dotted border-input" />
        )}
      </span>
      <input
        {...register("q", {
          onBlur: () => window.setTimeout(() => setOpen(false), 120),
        })}
        aria-label={label}
        role="combobox"
        aria-expanded={showList}
        aria-controls={listId}
        aria-autocomplete="list"
        autoComplete="off"
        placeholder={placeholder}
        onFocus={(e) => {
          setOpen(true);
          e.currentTarget.select();
        }}
        className={cn(
          "w-full h-10 px-4 rounded-full bg-surface-sunken text-sm text-foreground placeholder:text-subtle-foreground outline-none border border-transparent focus:border-ring focus:bg-card",
          value?.isMe && "text-sky-ink font-medium",
        )}
      />
      {showList && (
        <ul
          id={listId}
          role="listbox"
          className="col-start-2 mt-1.5 bg-card rounded-2xl shadow-soft-lg border border-border py-1.5 overflow-hidden"
        >
          {allowMe && !value?.isMe && (
            <li
              role="option"
              aria-selected={false}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                setOpen(false);
                onUseMyLocation?.();
              }}
              className="flex items-center gap-3 px-4 py-2.5 cursor-pointer hover:bg-muted text-sm text-sky-ink font-medium"
            >
              {locating ? <Loader2 className="size-4 animate-spin" /> : <LocateFixed className="size-4" />}
              Your location
            </li>
          )}
          {suggestions.items.map((p) => (
            <li
              key={p.place_id}
              role="option"
              aria-selected={false}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => choose({ label: p.name, lat: p.location.lat, lng: p.location.lng })}
              className="px-4 py-2 cursor-pointer hover:bg-muted"
            >
              <span className="block text-sm text-foreground truncate">{p.name}</span>
              {p.address && <span className="block text-xs text-muted-foreground truncate">{p.address}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function formatDistance(m: number) {
  return m >= 1000 ? `${(m / 1000).toFixed(m >= 10_000 ? 0 : 1)} km` : `${Math.round(m)} m`;
}

export function formatDuration(sec: number) {
  const mins = Math.max(1, Math.round(sec / 60));
  if (mins < 60) return `${mins} min`;
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return m ? `${h} hr ${m} min` : `${h} hr`;
}
