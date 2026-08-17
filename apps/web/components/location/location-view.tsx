"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  BookmarkPlus,
  Check,
  CloudSun,
  Loader2,
  MapPin,
  Navigation,
  Route as RouteIcon,
  Search,
  Star,
} from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import {
  errorMessage,
  useComputeRoute,
  useCreateSavedPlace,
  useDeleteSavedPlace,
  useNearbyPlaces,
  usePlaceDetails,
  useSavedPlaces,
  useSearchPlaces,
  useWeather,
} from "@/hooks/location-hooks";
import type {
  PlaceSummary,
  Route,
  SavedPlace,
  TravelMode,
} from "@/lib/apis/location-api";
import {
  routeSchema,
  type RouteFormInput,
} from "@/lib/schemas/location-schemas";

import type { MapPoint } from "./location-map";

const LocationMap = dynamic(() => import("./location-map"), { ssr: false });

const DEFAULT_CENTER = { lat: 25.033, lng: 121.5654 };

type Tab = "search" | "saved" | "route";

type Selection =
  | { kind: "place"; place: PlaceSummary }
  | { kind: "saved"; place: SavedPlace }
  | { kind: "coord"; lat: number; lng: number };

const MODE_OPTIONS = [
  { value: "driving", label: "Driving" },
  { value: "walking", label: "Walking" },
  { value: "bicycling", label: "Bicycling" },
  { value: "transit", label: "Transit" },
];

export function LocationView() {
  const [tab, setTab] = useState<Tab>("search");
  const [selection, setSelection] = useState<Selection | null>(null);
  const [center, setCenter] = useState<
    { lat: number; lng: number } | undefined
  >();
  const [showWeather, setShowWeather] = useState(false);
  const [route, setRoute] = useState<Route | null>(null);

  const saved = useSavedPlaces();
  const savedList = useMemo(
    () => (saved.data ?? []).filter((p) => p.category !== "weather"),
    [saved.data],
  );
  const del = useDeleteSavedPlace();
  const createSaved = useCreateSavedPlace();
  const savedPlaceIds = useMemo(() => {
    const s = new Set<string>();
    for (const p of savedList) if (p.place_id) s.add(p.place_id);
    return s;
  }, [savedList]);

  const selectedCoord = useMemo(() => coordOf(selection), [selection]);

  const weather = useWeather(
    showWeather && selectedCoord ? { ...selectedCoord, units: "metric" } : null,
  );

  const selectPlace = (place: PlaceSummary) => {
    setSelection({ kind: "place", place });
    setCenter(place.location);
    setShowWeather(false);
  };
  const selectSaved = (place: SavedPlace) => {
    setSelection({ kind: "saved", place });
    setCenter({ lat: place.lat, lng: place.lng });
    setShowWeather(false);
  };
  const selectCoord = (lat: number, lng: number) => {
    setSelection({ kind: "coord", lat, lng });
    setShowWeather(false);
  };

  const savedMatchForSelection: SavedPlace | undefined =
    selection?.kind === "saved"
      ? savedList.find((s) => s.id === selection.place.id)
      : selection?.kind === "place" && selection.place.place_id
        ? savedList.find((s) => s.place_id === selection.place.place_id)
        : undefined;

  const selectionIsSaved = !!savedMatchForSelection;

  const toggleSaveSelection = () => {
    if (!selection || createSaved.isPending || del.isPending) return;
    if (savedMatchForSelection) {
      del.mutate(savedMatchForSelection.id);
      return;
    }
    const onSaved = {
      onSuccess: (saved: SavedPlace) =>
        setSelection({ kind: "saved", place: saved }),
    };
    if (selection.kind === "place") {
      const p = selection.place;
      createSaved.mutate(
        {
          name: p.name,
          address: p.address || undefined,
          lat: p.location.lat,
          lng: p.location.lng,
          place_id: p.place_id || undefined,
        },
        onSaved,
      );
    } else if (selection.kind === "saved") {
      const p = selection.place;
      createSaved.mutate(
        {
          name: p.name,
          address: p.address || undefined,
          lat: p.lat,
          lng: p.lng,
          place_id: p.place_id || undefined,
        },
        onSaved,
      );
    } else if (selection.kind === "coord") {
      createSaved.mutate(
        { name: "Pinned point", lat: selection.lat, lng: selection.lng },
        onSaved,
      );
    }
  };

  const points = useMemo<MapPoint[]>(() => {
    const acc: MapPoint[] = [];
    const selectedSavedId =
      selection?.kind === "saved" ? selection.place.id : null;
    for (const s of savedList) {
      if (s.id === selectedSavedId) continue;
      acc.push({
        id: s.id,
        lat: s.lat,
        lng: s.lng,
        label: s.name,
        kind: "saved",
      });
    }
    if (selection?.kind === "saved") {
      acc.push({
        id: `sel-${selection.place.id}`,
        lat: selection.place.lat,
        lng: selection.place.lng,
        label: selection.place.name,
        kind: "saved",
      });
    } else if (selection?.kind === "place") {
      acc.push({
        id: selection.place.place_id,
        lat: selection.place.location.lat,
        lng: selection.place.location.lng,
        label: selection.place.name,
        kind: "selected",
      });
    } else if (selection?.kind === "coord") {
      acc.push({
        id: `${selection.lat}-${selection.lng}`,
        lat: selection.lat,
        lng: selection.lng,
        label: "Pinned",
        kind: "selected",
      });
    }
    if (route?.legs && route.legs.length > 0) {
      const first = route.legs[0].start_location;
      const last = route.legs[route.legs.length - 1].end_location;
      acc.push({
        id: "route-origin",
        lat: first.lat,
        lng: first.lng,
        kind: "origin",
        label: "Start",
      });
      acc.push({
        id: "route-dest",
        lat: last.lat,
        lng: last.lng,
        kind: "destination",
        label: "End",
      });
    }
    return acc;
  }, [savedList, selection, route]);

  const onMarkerClick = (p: MapPoint) => {
    if (p.kind === "saved") {
      const found = savedList.find((s) => s.id === p.id);
      if (found) selectSaved(found);
    }
  };

  return (
    <section className="flex flex-col gap-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1
            className="text-3xl tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              color: "#ECEFEA",
            }}
          >
            Places
          </h1>
          <p className="text-sm mt-1" style={{ color: "#8B9A8E" }}>
            Search, save, and plan around the world.
          </p>
        </div>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-[380px_1fr] gap-4 items-start">
        <div
          className="flex flex-col gap-3 rounded-2xl p-4"
          style={{
            backgroundColor: "#121814",
            border: "1px solid #1F2A24",
            minHeight: 520,
          }}
        >
          {selection && (
            <SelectionCard
              selection={selection}
              onSave={toggleSaveSelection}
              isSaved={selectionIsSaved}
              isSaving={createSaved.isPending}
              isUnsaving={selectionIsSaved && del.isPending}
              onWeather={() => setShowWeather((v) => !v)}
              weatherOpen={showWeather}
              weather={
                showWeather
                  ? {
                      isLoading: weather.isLoading,
                      error: weather.isError
                        ? errorMessage(weather.error)
                        : null,
                      hourly: weather.data?.hourly ?? [],
                      daily: weather.data?.daily ?? [],
                    }
                  : null
              }
            />
          )}

          <TabBar tab={tab} setTab={setTab} />

          {tab === "search" && (
            <SearchTab near={selectedCoord} onPick={selectPlace} />
          )}
          {tab === "saved" && (
            <SavedTab
              places={savedList}
              isLoading={saved.isLoading}
              error={saved.isError ? errorMessage(saved.error) : null}
              onPick={selectSaved}
            />
          )}
          {tab === "route" && (
            <RouteTab
              onResult={(r) => setRoute(r)}
              seedFrom={selectedCoord}
              route={route}
            />
          )}
        </div>

        <div
          className="rounded-2xl overflow-hidden sticky top-4"
          style={{ height: 520, border: "1px solid #1F2A24" }}
        >
          <LocationMap
            points={points}
            polyline={route?.polyline}
            center={center ?? DEFAULT_CENTER}
            zoom={12}
            onMapClick={selectCoord}
            onMarkerClick={onMarkerClick}
          />
        </div>
      </div>
    </section>
  );
}

function TabBar({ tab, setTab }: { tab: Tab; setTab: (t: Tab) => void }) {
  const cls = (t: Tab) =>
    `flex-1 text-sm px-3 py-1.5 rounded-full ${
      tab === t
        ? "bg-white/10 text-[#ECEFEA]"
        : "text-[#8B9A8E] hover:bg-white/5"
    }`;
  return (
    <div className="flex items-center gap-1 p-1 rounded-full bg-[#0B100D] border border-[#1F2A24]">
      <button
        type="button"
        onClick={() => setTab("search")}
        className={cls("search")}
      >
        Search
      </button>
      <button
        type="button"
        onClick={() => setTab("saved")}
        className={cls("saved")}
      >
        Saved
      </button>
      <button
        type="button"
        onClick={() => setTab("route")}
        className={cls("route")}
      >
        Route
      </button>
    </div>
  );
}

function SearchTab({
  near,
  onPick,
}: {
  near?: { lat: number; lng: number };
  onPick: (p: PlaceSummary) => void;
}) {
  const [draft, setDraft] = useState("");
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    const t = setTimeout(() => setDebounced(draft.trim()), 250);
    return () => clearTimeout(t);
  }, [draft]);

  const search = useSearchPlaces({
    q: debounced,
    ...(near ? { lat: near.lat, lng: near.lng, radius: 5000 } : {}),
    limit: 12,
  });
  const nearby = useNearbyPlaces(
    near ? { ...near, radius: 1500, limit: 12 } : null,
  );

  return (
    <div className="flex flex-col gap-3 min-h-0">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 size-4 text-[#8B9A8E]" />
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Search places (e.g. sushi, museum)"
          className="w-full h-9 pl-9 pr-9 rounded-lg bg-[#0B100D] text-sm text-[#ECEFEA] border border-[#1F2A24] focus:outline-none focus:border-white/20"
        />
        {search.isFetching && debounced && (
          <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 size-4 animate-spin text-[#8B9A8E]" />
        )}
      </div>

      {debounced && (
        <PlaceList
          title="Search results"
          items={search.data ?? []}
          isLoading={search.isLoading}
          error={search.isError ? errorMessage(search.error) : null}
          empty="No matches."
          onPick={onPick}
        />
      )}

      {near ? (
        <PlaceList
          title="Nearby"
          items={nearby.data ?? []}
          isLoading={nearby.isLoading}
          error={nearby.isError ? errorMessage(nearby.error) : null}
          empty="Nothing nearby."
          onPick={onPick}
        />
      ) : (
        <p className="text-xs text-[#6B7A6F]">
          Click the map or pick a place to see what&apos;s nearby.
        </p>
      )}
    </div>
  );
}

function PlaceList({
  title,
  items,
  isLoading,
  error,
  empty,
  onPick,
}: {
  title: string;
  items: PlaceSummary[];
  isLoading: boolean;
  error: string | null;
  empty: string;
  onPick: (p: PlaceSummary) => void;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <p
        className="text-[11px] uppercase tracking-wider"
        style={{ color: "var(--season-button)" }}
      >
        {title}
      </p>
      {isLoading && <p className="text-xs text-[#8B9A8E]">Loading…</p>}
      {error && <p className="text-xs text-[#FCA5A5]">{error}</p>}
      {!isLoading && !error && items.length === 0 && (
        <p className="text-xs text-[#6B7A6F]">{empty}</p>
      )}
      <ul className="flex flex-col gap-1.5">
        {items.map((p) => (
          <li key={p.place_id}>
            <button
              type="button"
              onClick={() => onPick(p)}
              className="w-full text-left px-3 py-2 rounded-lg hover:bg-white/5 border border-[#1F2A24]"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-sm text-[#ECEFEA] truncate">{p.name}</p>
                  {p.address && (
                    <p className="text-xs text-[#8B9A8E] truncate">
                      {p.address}
                    </p>
                  )}
                </div>
                {p.rating !== undefined && (
                  <span className="text-xs text-[#F3B664] shrink-0 inline-flex items-center gap-0.5">
                    <Star className="size-3" />
                    {p.rating.toFixed(1)}
                  </span>
                )}
              </div>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function SavedTab({
  places,
  isLoading,
  error,
  onPick,
}: {
  places: SavedPlace[];
  isLoading: boolean;
  error: string | null;
  onPick: (p: SavedPlace) => void;
}) {
  return (
    <div className="flex flex-col gap-2 min-h-0">
      {isLoading && <p className="text-xs text-[#8B9A8E]">Loading…</p>}
      {error && <p className="text-xs text-[#FCA5A5]">{error}</p>}
      {!isLoading && !error && places.length === 0 && (
        <p className="text-xs text-[#6B7A6F]">
          You haven&apos;t saved any places yet. Search or pin one on the map.
        </p>
      )}
      <ul className="flex flex-col gap-1.5">
        {places.map((p) => (
          <li key={p.id}>
            <button
              type="button"
              onClick={() => onPick(p)}
              className="w-full text-left px-3 py-2 rounded-lg hover:bg-white/5 border border-[#1F2A24]"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-sm text-[#ECEFEA] truncate">{p.name}</p>
                  {p.address && (
                    <p className="text-xs text-[#8B9A8E] truncate">
                      {p.address}
                    </p>
                  )}
                </div>
                {p.category && (
                  <span className="text-[10px] uppercase tracking-wider text-[#8B9A8E] shrink-0">
                    {p.category}
                  </span>
                )}
              </div>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function RouteTab({
  onResult,
  seedFrom,
  route,
}: {
  onResult: (r: Route) => void;
  seedFrom?: { lat: number; lng: number };
  route: Route | null;
}) {
  const compute = useComputeRoute();
  const form = useForm<RouteFormInput>({
    resolver: zodResolver(routeSchema),
    defaultValues: {
      origin_lat: seedFrom?.lat ?? 0,
      origin_lng: seedFrom?.lng ?? 0,
      destination_lat: 0,
      destination_lng: 0,
      mode: "driving",
    },
  });

  const onSubmit = (v: RouteFormInput) => {
    compute.mutate(
      {
        origin: { lat: v.origin_lat, lng: v.origin_lng },
        destination: { lat: v.destination_lat, lng: v.destination_lng },
        mode: v.mode as TravelMode,
      },
      { onSuccess: onResult },
    );
  };

  const err = errorMessage(compute.error);

  return (
    <div className="flex flex-col gap-3 min-h-0">
      <FormProvider {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
          className="flex flex-col gap-3"
        >
          <div className="grid grid-cols-2 gap-2">
            <ControlledInput<RouteFormInput>
              name="origin_lat"
              label="Origin lat"
              type="number"
              step="any"
              valueAsNumber
            />
            <ControlledInput<RouteFormInput>
              name="origin_lng"
              label="Origin lng"
              type="number"
              step="any"
              valueAsNumber
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <ControlledInput<RouteFormInput>
              name="destination_lat"
              label="Dest lat"
              type="number"
              step="any"
              valueAsNumber
            />
            <ControlledInput<RouteFormInput>
              name="destination_lng"
              label="Dest lng"
              type="number"
              step="any"
              valueAsNumber
            />
          </div>
          <ControlledSelect<RouteFormInput>
            name="mode"
            label="Mode"
            options={MODE_OPTIONS}
          />
          <button
            type="submit"
            disabled={compute.isPending}
            className="season-transition inline-flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D] self-start"
            style={{ backgroundColor: "var(--season-button)" }}
          >
            {compute.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <RouteIcon className="size-4" />
            )}
            Compute route
          </button>
        </form>
      </FormProvider>

      {err && <p className="text-xs text-[#FCA5A5]">{err}</p>}
      {route && (
        <div className="rounded-lg border border-[#1F2A24] px-3 py-2 text-xs text-[#ECEFEA]">
          <p>
            <Navigation className="inline size-3 mr-1 text-[#F3B664]" />
            {formatDistance(route.distance_meters)} ·{" "}
            {formatDuration(route.duration_seconds)}
          </p>
          {route.summary && (
            <p className="text-[#8B9A8E] mt-1">{route.summary}</p>
          )}
        </div>
      )}
    </div>
  );
}

function SelectionCard({
  selection,
  onSave,
  isSaved,
  isSaving,
  isUnsaving,
  onWeather,
  weatherOpen,
  weather,
}: {
  selection: Selection;
  onSave: () => void;
  isSaved: boolean;
  isSaving: boolean;
  isUnsaving: boolean;
  onWeather: () => void;
  weatherOpen: boolean;
  weather: {
    isLoading: boolean;
    error: string | null;
    hourly: {
      time: string;
      temp_c: number;
      icon?: string;
      summary?: string;
      pop_pct?: number;
    }[];
    daily: {
      date: string;
      temp_min_c: number;
      temp_max_c: number;
      icon?: string;
      summary?: string;
      pop_pct?: number;
    }[];
  } | null;
}) {
  const coord = coordOf(selection);
  const placeId =
    selection.kind === "place" ? selection.place.place_id : undefined;
  const details = usePlaceDetails(placeId);

  const header =
    selection.kind === "saved"
      ? selection.place.name
      : selection.kind === "place"
        ? selection.place.name
        : "Pinned point";
  const address =
    selection.kind === "saved"
      ? selection.place.address
      : selection.kind === "place"
        ? selection.place.address
        : undefined;

  return (
    <div
      className="rounded-xl p-3 flex flex-col gap-2"
      style={{ backgroundColor: "#0B100D", border: "1px solid #1F2A24" }}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm text-[#ECEFEA] truncate">{header}</p>
          {address && (
            <p className="text-xs text-[#8B9A8E] truncate">{address}</p>
          )}
          {coord && (
            <p className="text-[11px] text-[#6B7A6F] mt-0.5">
              <MapPin className="inline size-3 mr-0.5" />
              {coord.lat.toFixed(4)}, {coord.lng.toFixed(4)}
            </p>
          )}
        </div>
      </div>

      {selection.kind === "place" && details.data?.website && (
        <a
          href={details.data.website}
          target="_blank"
          rel="noreferrer noopener"
          className="text-xs text-[#79B4E6] hover:underline truncate"
        >
          {details.data.website}
        </a>
      )}
      {selection.kind === "place" && details.data?.phone_number && (
        <p className="text-xs text-[#8B9A8E]">{details.data.phone_number}</p>
      )}
      {selection.kind === "place" &&
        details.data?.opening_hours &&
        details.data.opening_hours.length > 0 && (
          <details className="text-xs text-[#8B9A8E]">
            <summary className="cursor-pointer">Hours</summary>
            <ul className="mt-1 pl-3">
              {details.data.opening_hours.map((h, i) => (
                <li key={i}>{h}</li>
              ))}
            </ul>
          </details>
        )}

      <div className="flex flex-wrap items-center gap-1.5 pt-1">
        {coord && (
          <button
            type="button"
            onClick={onSave}
            disabled={isSaving || isUnsaving}
            className="inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full hover:bg-white/5 disabled:hover:bg-transparent disabled:opacity-80 text-[#ECEFEA]"
          >
            {isSaving || isUnsaving ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : isSaved ? (
              <Check
                className="size-3.5"
                style={{ color: "var(--season-button)" }}
              />
            ) : (
              <BookmarkPlus className="size-3.5" />
            )}
            {isSaving
              ? "Saving…"
              : isUnsaving
                ? "Removing…"
                : isSaved
                  ? "Saved"
                  : "Save"}
          </button>
        )}
        <button
          type="button"
          onClick={onWeather}
          className={`inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full hover:bg-white/5 text-[#ECEFEA] ${
            weatherOpen ? "bg-white/5" : ""
          }`}
        >
          <CloudSun className="size-3.5" />
          {weatherOpen ? "Hide weather" : "Weather"}
        </button>
      </div>

      {selection.kind === "saved" && selection.place.notes && (
        <p className="text-xs whitespace-pre-wrap text-[#8B9A8E] pt-1 border-t border-[#1F2A24] mt-1">
          {selection.place.notes}
        </p>
      )}

      {weather && (
        <WeatherBottomBar
          isLoading={weather.isLoading}
          error={weather.error}
          hourly={weather.hourly}
          daily={weather.daily}
        />
      )}
    </div>
  );
}

function WeatherBottomBar({
  isLoading,
  error,
  hourly,
  daily,
}: {
  isLoading: boolean;
  error: string | null;
  hourly: {
    time: string;
    temp_c: number;
    icon?: string;
    summary?: string;
    pop_pct?: number;
  }[];
  daily: {
    date: string;
    temp_min_c: number;
    temp_max_c: number;
    icon?: string;
    summary?: string;
    pop_pct?: number;
  }[];
}) {
  const ref = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    ref.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, []);
  return (
    <div
      ref={ref}
      className="mt-2 -mx-3 -mb-3 rounded-b-xl px-3 py-2.5 border-t flex flex-col gap-2.5"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      {error && <p className="text-xs text-[#FCA5A5]">{error}</p>}
      {!error && hourly.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <p
              className="text-[10px] font-medium tracking-[0.18em] uppercase"
              style={{ color: "var(--season-button)" }}
            >
              Today · hourly
            </p>
            {isLoading && (
              <Loader2 className="size-3 animate-spin text-[#8B9A8E]" />
            )}
          </div>
          <ul className="flex gap-1.5 overflow-x-auto pb-1 -mx-1 px-1">
            {hourly.map((h) => (
              <li
                key={h.time}
                className="flex flex-col items-center gap-0.5 rounded-lg px-2 py-1.5 border shrink-0"
                style={{
                  backgroundColor: "#0B100D",
                  borderColor: "#1F2A24",
                  minWidth: 56,
                }}
              >
                <span className="text-[10px] tracking-wider text-[#8B9A8E]">
                  {shortHour(h.time)}
                </span>
                {h.icon ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={h.icon}
                    alt={h.summary ?? ""}
                    width={28}
                    height={28}
                    loading="lazy"
                    style={{ width: 28, height: 28 }}
                  />
                ) : (
                  <span className="h-7" />
                )}
                <span className="text-[11px] text-[#ECEFEA]">
                  {Math.round(h.temp_c)}°
                </span>
                {h.pop_pct !== undefined && h.pop_pct > 0 && (
                  <span className="text-[10px] text-[#79B4E6]">
                    {Math.round(h.pop_pct)}%
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      {!error && daily.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <p
              className="text-[10px] font-medium tracking-[0.18em] uppercase"
              style={{ color: "var(--season-button)" }}
            >
              5-day
            </p>
            {isLoading && hourly.length === 0 && (
              <Loader2 className="size-3 animate-spin text-[#8B9A8E]" />
            )}
          </div>
          <ul className="grid grid-flow-col auto-cols-fr gap-1.5">
            {daily.slice(0, 5).map((d) => (
              <li
                key={d.date}
                className="flex flex-col items-center gap-0.5 rounded-lg px-1.5 py-1.5 border"
                style={{
                  backgroundColor: "#0B100D",
                  borderColor: "#1F2A24",
                }}
              >
                <span className="text-[10px] uppercase tracking-wider text-[#8B9A8E]">
                  {shortDate(d.date)}
                </span>
                {d.icon ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={d.icon}
                    alt={d.summary ?? ""}
                    width={28}
                    height={28}
                    loading="lazy"
                    style={{ width: 28, height: 28 }}
                  />
                ) : (
                  <span className="h-7" />
                )}
                <span className="text-[11px] text-[#ECEFEA]">
                  {Math.round(d.temp_max_c)}°
                  <span className="text-[#8B9A8E]">
                    /{Math.round(d.temp_min_c)}°
                  </span>
                </span>
                {d.pop_pct !== undefined && (
                  <span className="text-[10px] text-[#79B4E6]">
                    {Math.round(d.pop_pct)}%
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      {isLoading && daily.length === 0 && hourly.length === 0 && !error && (
        <p className="text-xs text-[#8B9A8E] flex items-center gap-1.5">
          <Loader2 className="size-3 animate-spin" />
          Loading forecast…
        </p>
      )}
    </div>
  );
}

function coordOf(
  sel: Selection | null,
): { lat: number; lng: number } | undefined {
  if (!sel) return undefined;
  if (sel.kind === "place") return sel.place.location;
  if (sel.kind === "saved") return { lat: sel.place.lat, lng: sel.place.lng };
  return { lat: sel.lat, lng: sel.lng };
}

function formatDistance(m: number): string {
  if (m < 1000) return `${Math.round(m)} m`;
  return `${(m / 1000).toFixed(m < 10_000 ? 1 : 0)} km`;
}

function formatDuration(sec: number): string {
  const mins = Math.round(sec / 60);
  if (mins < 60) return `${mins} min`;
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return m === 0 ? `${h} h` : `${h} h ${m} min`;
}

function shortDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { weekday: "short", day: "numeric" });
}

function shortHour(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, { hour: "numeric" });
}
