"use client";

import { useMemo, useState } from "react";
import { Bookmark, BookmarkMinus, Loader2, MapPin, X } from "lucide-react";

import {
  PlaceSearchPicker,
  type PickedPlace,
} from "@/components/trips/place-search-picker";
import {
  errorMessage,
  useCreateSavedPlace,
  useDeleteSavedPlace,
  useSavedPlaces,
  useWeather,
} from "@/hooks/location-hooks";
import type {
  SavedPlace,
  WeatherDay,
  WeatherPoint,
} from "@/lib/apis/location-api";

const WEATHER_CATEGORY = "weather";

type Tab = "today" | "7" | "14" | "30";

const TAB_DAYS: Record<Exclude<Tab, "today">, number> = {
  "7": 7,
  "14": 14,
  "30": 30,
};

export function WeatherView() {
  const [picked, setPicked] = useState<PickedPlace | null>(null);
  const [tab, setTab] = useState<Tab>("today");

  const savedQuery = useSavedPlaces({ category: WEATHER_CATEGORY });
  const savedList = useMemo<SavedPlace[]>(
    () => savedQuery.data ?? [],
    [savedQuery.data],
  );

  const weather = useWeather(
    picked ? { lat: picked.lat, lng: picked.lng, units: "metric" } : null,
    { refetchOnWindowFocus: true, staleTime: 5 * 60_000 },
  );
  const save = useCreateSavedPlace();
  const del = useDeleteSavedPlace();

  const daily = useMemo<WeatherDay[]>(
    () => weather.data?.daily ?? [],
    [weather.data?.daily],
  );
  const hourly = useMemo<WeatherPoint[]>(
    () => weather.data?.hourly ?? [],
    [weather.data?.hourly],
  );

  const savedMatch = useMemo<SavedPlace | undefined>(() => {
    if (!picked) return undefined;
    if (picked.place_id) {
      const m = savedList.find((s) => s.place_id === picked.place_id);
      if (m) return m;
    }
    return savedList.find(
      (s) =>
        Math.abs(s.lat - picked.lat) < 1e-4 &&
        Math.abs(s.lng - picked.lng) < 1e-4,
    );
  }, [picked, savedList]);

  const handlePick = (p: PickedPlace) => {
    setPicked(p);
  };

  const handlePickSaved = (s: SavedPlace) => {
    setPicked({
      name: s.name,
      address: s.address ?? "",
      lat: s.lat,
      lng: s.lng,
      place_id: s.place_id ?? "",
    });
  };

  const handleToggleSave = () => {
    if (!picked || save.isPending || del.isPending) return;
    if (savedMatch) {
      del.mutate(savedMatch.id);
      return;
    }
    save.mutate({
      name: picked.name,
      address: picked.address,
      lat: picked.lat,
      lng: picked.lng,
      place_id: picked.place_id || undefined,
      category: WEATHER_CATEGORY,
    });
  };

  const handleRemoveSaved = (id: string) => {
    if (del.isPending) return;
    del.mutate(id);
  };

  const mutationError = errorMessage(save.error ?? del.error);
  const toggleBusy = save.isPending || del.isPending;

  return (
    <section className="flex flex-col gap-6">
      <header className="flex flex-col gap-2">
        <p
          className="text-[11px] font-medium tracking-[0.2em] uppercase"
          style={{ color: "var(--season-button)" }}
        >
          Weather
        </p>
        <h1
          className="text-3xl tracking-tight text-[#ECEFEA]"
          style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
        >
          Check the sky before you go
        </h1>
        <p className="text-sm text-[#8B9A8E]">
          Search a place to see today&apos;s conditions and the next 30 days.
        </p>
      </header>

      <div className="flex flex-col gap-2">
        <PlaceSearchPicker
          onPick={handlePick}
          placeholder="Search a city or place"
        />
        {picked && (
          <div className="flex items-center gap-2 text-sm text-[#8B9A8E]">
            <MapPin className="size-4" />
            <span className="text-[#ECEFEA]">{picked.name}</span>
            {picked.address && (
              <span className="truncate">· {picked.address}</span>
            )}
            <button
              type="button"
              onClick={handleToggleSave}
              disabled={toggleBusy}
              className="ml-auto inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full border border-[#1F2A24] hover:bg-white/5 disabled:opacity-60 text-[#ECEFEA]"
              style={
                savedMatch ? { borderColor: "var(--season-button)" } : undefined
              }
            >
              {toggleBusy ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : savedMatch ? (
                <BookmarkMinus
                  className="size-3.5"
                  style={{ color: "var(--season-button)" }}
                />
              ) : (
                <Bookmark className="size-3.5" />
              )}
              {save.isPending
                ? "Saving…"
                : del.isPending
                  ? "Removing…"
                  : savedMatch
                    ? "Saved"
                    : "Save place"}
            </button>
          </div>
        )}
        {mutationError && (
          <p className="text-xs text-[#FCA5A5]">{mutationError}</p>
        )}
      </div>

      <SavedWeatherList
        places={savedList}
        loading={savedQuery.isLoading}
        activeId={savedMatch?.id}
        onPick={handlePickSaved}
        onRemove={handleRemoveSaved}
        removingId={del.isPending ? (del.variables ?? null) : null}
      />

      {!picked ? (
        <EmptyState />
      ) : weather.isLoading ? (
        <LoadingState />
      ) : weather.isError ? (
        <ErrorState
          message={errorMessage(weather.error) ?? "Failed to load forecast."}
        />
      ) : (
        <>
          <Tabs value={tab} onChange={setTab} />
          {tab === "today" ? (
            <TodayPanel
              current={weather.data?.current}
              first={daily[0]}
              hourly={hourly}
            />
          ) : (
            <DailyListPanel days={daily} limit={TAB_DAYS[tab]} />
          )}
        </>
      )}
    </section>
  );
}

function Tabs({ value, onChange }: { value: Tab; onChange: (t: Tab) => void }) {
  const items: { key: Tab; label: string }[] = [
    { key: "today", label: "Today" },
    { key: "7", label: "7 days" },
    { key: "14", label: "14 days" },
    { key: "30", label: "30 days" },
  ];
  return (
    <div className="flex items-center gap-1 border-b border-[#1F2A24]">
      {items.map((it) => {
        const active = it.key === value;
        return (
          <button
            key={it.key}
            type="button"
            onClick={() => onChange(it.key)}
            className="px-4 py-2 text-sm border-b-2 -mb-px"
            style={{
              borderColor: active ? "var(--season-button)" : "transparent",
              color: active ? "#ECEFEA" : "#8B9A8E",
            }}
          >
            {it.label}
          </button>
        );
      })}
    </div>
  );
}

function TodayPanel({
  current,
  first,
  hourly,
}: {
  current?: WeatherPoint;
  first?: WeatherDay;
  hourly: WeatherPoint[];
}) {
  if (!current && !first && hourly.length === 0) {
    return <p className="text-sm text-[#8B9A8E]">No forecast available.</p>;
  }
  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-4 md:grid-cols-[1.2fr_1fr]">
        {current && (
          <div className="rounded-2xl border border-[#1F2A24] bg-[#121814] p-6 flex flex-col gap-3">
            <p className="text-xs uppercase tracking-widest text-[#8B9A8E]">
              Now
            </p>
            <div className="flex items-center gap-3">
              {current.icon && (
                <WeatherIcon
                  src={current.icon}
                  alt={current.summary ?? ""}
                  size={64}
                />
              )}
              <span className="text-5xl text-[#ECEFEA]">
                {Math.round(current.temp_c)}°
              </span>
              {current.summary && (
                <span className="text-sm text-[#8B9A8E] capitalize">
                  {current.summary}
                </span>
              )}
            </div>
            <div className="grid grid-cols-3 gap-3 text-xs text-[#8B9A8E]">
              {current.feels_like_c !== undefined && (
                <Stat
                  label="Feels like"
                  value={`${Math.round(current.feels_like_c)}°`}
                />
              )}
              {current.humidity !== undefined && (
                <Stat label="Humidity" value={`${current.humidity}%`} />
              )}
              {current.wind_kph !== undefined && (
                <Stat
                  label="Wind"
                  value={`${Math.round(current.wind_kph)} km/h`}
                />
              )}
              {current.pop_pct !== undefined && (
                <Stat
                  label="Precip."
                  value={`${Math.round(current.pop_pct)}%`}
                />
              )}
            </div>
          </div>
        )}
        {first && (
          <div className="rounded-2xl border border-[#1F2A24] bg-[#121814] p-6 flex flex-col gap-3">
            <p className="text-xs uppercase tracking-widest text-[#8B9A8E]">
              {formatDay(first.date)}
            </p>
            <div className="flex items-center gap-3">
              {first.icon && (
                <WeatherIcon
                  src={first.icon}
                  alt={first.summary ?? ""}
                  size={56}
                />
              )}
              <div className="flex items-baseline gap-2">
                <span className="text-4xl text-[#ECEFEA]">
                  {Math.round(first.temp_max_c)}°
                </span>
                <span className="text-lg text-[#8B9A8E]">
                  / {Math.round(first.temp_min_c)}°
                </span>
              </div>
            </div>
            {first.summary && (
              <p className="text-sm text-[#ECEFEA] capitalize">
                {first.summary}
              </p>
            )}
            {first.pop_pct !== undefined && (
              <p className="text-xs text-[#8B9A8E]">
                Precip. chance {Math.round(first.pop_pct)}%
              </p>
            )}
          </div>
        )}
      </div>
      {hourly.length > 0 && <HourlyStrip hourly={hourly} />}
    </div>
  );
}

function HourlyStrip({ hourly }: { hourly: WeatherPoint[] }) {
  return (
    <div className="rounded-2xl border border-[#1F2A24] bg-[#121814] p-4 flex flex-col gap-3">
      <p className="text-xs uppercase tracking-widest text-[#8B9A8E]">
        Next hours
      </p>
      <ul className="hourly-scroll flex gap-2 overflow-x-auto pb-1 -mx-1 px-1">
        {hourly.map((h) => (
          <li
            key={h.time}
            className="flex flex-col items-center gap-1 rounded-xl border border-[#1F2A24] bg-[#0E1411] px-3 py-2 shrink-0"
            style={{ minWidth: 72 }}
          >
            <span className="text-xs text-[#8B9A8E]">{formatHour(h.time)}</span>
            {h.icon ? (
              <WeatherIcon src={h.icon} alt={h.summary ?? ""} size={36} />
            ) : (
              <span className="text-[#6B7A6F]">—</span>
            )}
            <span className="text-sm text-[#ECEFEA]">
              {Math.round(h.temp_c)}°
            </span>
            {h.pop_pct !== undefined && h.pop_pct > 0 && (
              <span className="text-[10px] text-[#8B9A8E]">
                {Math.round(h.pop_pct)}%
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

function DailyListPanel({
  days,
  limit,
}: {
  days: WeatherDay[];
  limit: number;
}) {
  if (days.length === 0) {
    return (
      <p className="text-sm text-[#8B9A8E]">No daily forecast available.</p>
    );
  }
  return (
    <ul className="grid gap-2">
      {days.slice(0, limit).map((d) => (
        <li
          key={d.date}
          className="grid grid-cols-[110px_40px_1fr_120px] gap-3 items-center rounded-xl border border-[#1F2A24] bg-[#121814] px-4 py-3"
        >
          <span className="text-sm text-[#ECEFEA]">{formatDay(d.date)}</span>
          <div className="flex items-center justify-center">
            {d.icon ? (
              <WeatherIcon src={d.icon} alt={d.summary ?? ""} size={36} />
            ) : (
              <span className="text-[#6B7A6F]">—</span>
            )}
          </div>
          <span className="text-sm text-[#8B9A8E] truncate capitalize">
            {d.summary ?? "—"}
            {d.pop_pct !== undefined && ` · ${Math.round(d.pop_pct)}%`}
          </span>
          <span className="text-sm text-[#ECEFEA] text-right">
            {Math.round(d.temp_max_c)}°{" "}
            <span className="text-[#8B9A8E]">
              / {Math.round(d.temp_min_c)}°
            </span>
          </span>
        </li>
      ))}
    </ul>
  );
}

function WeatherIcon({
  src,
  alt,
  size,
}: {
  src: string;
  alt: string;
  size: number;
}) {
  return (
    /* OpenWeather icons are served over their CDN; plain img avoids Next image
       remote-pattern config for a small (<10 KB) sprite. */
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={src}
      alt={alt}
      width={size}
      height={size}
      loading="lazy"
      className="shrink-0"
      style={{ width: size, height: size }}
    />
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span>{label}</span>
      <span className="text-sm text-[#ECEFEA]">{value}</span>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="rounded-2xl border border-dashed border-[#1F2A24] p-10 text-center">
      <p className="text-sm text-[#8B9A8E]">
        Pick a place above to see the forecast.
      </p>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="rounded-2xl border border-[#1F2A24] p-10 flex items-center justify-center">
      <Loader2 className="size-4 animate-spin text-[#8B9A8E]" />
    </div>
  );
}

function SavedWeatherList({
  places,
  loading,
  activeId,
  onPick,
  onRemove,
  removingId,
}: {
  places: SavedPlace[];
  loading: boolean;
  activeId?: string;
  onPick: (p: SavedPlace) => void;
  onRemove: (id: string) => void;
  removingId: string | null;
}) {
  if (loading && places.length === 0) {
    return (
      <div className="flex items-center gap-2 text-xs text-[#8B9A8E]">
        <Loader2 className="size-3.5 animate-spin" />
        Loading saved places…
      </div>
    );
  }
  if (places.length === 0) return null;
  return (
    <div className="flex flex-col gap-2">
      <p className="text-[11px] font-medium tracking-[0.2em] uppercase text-[#8B9A8E]">
        Saved for weather
      </p>
      <ul className="flex flex-wrap gap-2">
        {places.map((p) => {
          const active = p.id === activeId;
          const removing = p.id === removingId;
          return (
            <li key={p.id}>
              <div
                className="inline-flex items-center gap-1 rounded-full border border-[#1F2A24] bg-[#121814] pl-3 pr-1 py-1 text-xs text-[#ECEFEA]"
                style={
                  active ? { borderColor: "var(--season-button)" } : undefined
                }
              >
                <button
                  type="button"
                  onClick={() => onPick(p)}
                  className="inline-flex items-center gap-1 hover:text-white"
                >
                  <MapPin className="size-3" />
                  <span>{p.name}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onRemove(p.id)}
                  disabled={removing}
                  className="ml-1 inline-flex items-center justify-center size-5 rounded-full hover:bg-white/10 disabled:opacity-60"
                  aria-label={`Remove ${p.name}`}
                >
                  {removing ? (
                    <Loader2 className="size-3 animate-spin" />
                  ) : (
                    <X className="size-3" />
                  )}
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function ErrorState({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-[#1F2A24] p-6">
      <p className="text-sm text-[#FCA5A5]">{message}</p>
    </div>
  );
}

function formatDay(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}

function formatHour(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, { hour: "numeric" });
}
