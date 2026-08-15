"use client";

import { useMemo, useState } from "react";
import { Bookmark, Check, Loader2, MapPin } from "lucide-react";

import {
  PlaceSearchPicker,
  type PickedPlace,
} from "@/components/trips/place-search-picker";
import {
  errorMessage,
  useCreateSavedPlace,
  useWeather,
} from "@/hooks/location-hooks";
import type { WeatherDay, WeatherPoint } from "@/lib/apis/location-api";

type Tab = "today" | "7" | "14" | "30";

const TAB_DAYS: Record<Exclude<Tab, "today">, number> = {
  "7": 7,
  "14": 14,
  "30": 30,
};

export function WeatherView() {
  const [picked, setPicked] = useState<PickedPlace | null>(null);
  const [tab, setTab] = useState<Tab>("today");
  const [saved, setSaved] = useState(false);

  const weather = useWeather(
    picked ? { lat: picked.lat, lng: picked.lng, units: "metric" } : null,
    { refetchOnWindowFocus: true, staleTime: 5 * 60_000 },
  );
  const save = useCreateSavedPlace({
    onSuccess: () => setSaved(true),
  });

  const daily = useMemo<WeatherDay[]>(
    () => weather.data?.daily ?? [],
    [weather.data?.daily],
  );

  const handlePick = (p: PickedPlace) => {
    setPicked(p);
    setSaved(false);
  };

  const handleSave = () => {
    if (!picked || save.isPending) return;
    save.mutate({
      name: picked.name,
      address: picked.address,
      lat: picked.lat,
      lng: picked.lng,
      place_id: picked.place_id || undefined,
      category: "weather",
    });
  };

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
              onClick={handleSave}
              disabled={save.isPending || saved}
              className="ml-auto inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full border border-[#1F2A24] hover:bg-white/5 disabled:opacity-60 text-[#ECEFEA]"
            >
              {save.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : saved ? (
                <Check className="size-3.5" style={{ color: "var(--season-button)" }} />
              ) : (
                <Bookmark className="size-3.5" />
              )}
              {saved ? "Saved" : "Save place"}
            </button>
          </div>
        )}
        {save.isError && (
          <p className="text-xs text-[#FCA5A5]">
            {errorMessage(save.error) ?? "Save failed."}
          </p>
        )}
      </div>

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
            <TodayPanel current={weather.data?.current} first={daily[0]} />
          ) : (
            <DailyListPanel days={daily} limit={TAB_DAYS[tab]} />
          )}
        </>
      )}
    </section>
  );
}

function Tabs({
  value,
  onChange,
}: {
  value: Tab;
  onChange: (t: Tab) => void;
}) {
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
}: {
  current?: WeatherPoint;
  first?: WeatherDay;
}) {
  if (!current && !first) {
    return <p className="text-sm text-[#8B9A8E]">No forecast available.</p>;
  }
  return (
    <div className="grid gap-4 md:grid-cols-[1.2fr_1fr]">
      {current && (
        <div className="rounded-2xl border border-[#1F2A24] bg-[#121814] p-6 flex flex-col gap-3">
          <p className="text-xs uppercase tracking-widest text-[#8B9A8E]">
            Now
          </p>
          <div className="flex items-center gap-3">
            {current.icon && (
              <WeatherIcon src={current.icon} alt={current.summary ?? ""} size={64} />
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
              <Stat label="Feels like" value={`${Math.round(current.feels_like_c)}°`} />
            )}
            {current.humidity !== undefined && (
              <Stat label="Humidity" value={`${current.humidity}%`} />
            )}
            {current.wind_kph !== undefined && (
              <Stat label="Wind" value={`${Math.round(current.wind_kph)} km/h`} />
            )}
            {current.pop_pct !== undefined && (
              <Stat label="Precip." value={`${Math.round(current.pop_pct)}%`} />
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
              <WeatherIcon src={first.icon} alt={first.summary ?? ""} size={56} />
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
            <p className="text-sm text-[#ECEFEA] capitalize">{first.summary}</p>
          )}
          {first.pop_pct !== undefined && (
            <p className="text-xs text-[#8B9A8E]">
              Precip. chance {Math.round(first.pop_pct)}%
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function DailyListPanel({ days, limit }: { days: WeatherDay[]; limit: number }) {
  if (days.length === 0) {
    return <p className="text-sm text-[#8B9A8E]">No daily forecast available.</p>;
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
            <span className="text-[#8B9A8E]">/ {Math.round(d.temp_min_c)}°</span>
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
