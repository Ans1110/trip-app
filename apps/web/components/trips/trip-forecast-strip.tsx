"use client";

import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { useWeather } from "@/hooks/location-hooks";
import type { WeatherDay } from "@/lib/apis/location-api";
import type { Trip } from "@/lib/apis/trip-api";

const FORECAST_WINDOW_DAYS = 30;

export function TripForecastStrip({ trip }: { trip: Trip }) {
  const hasCoord =
    trip.latitude !== undefined &&
    trip.longitude !== undefined &&
    trip.latitude !== null &&
    trip.longitude !== null;

  const daysUntilStart = useMemo(
    () => daysBetween(new Date(), parseDate(trip.start_date)),
    [trip.start_date],
  );
  const outOfWindow = daysUntilStart > FORECAST_WINDOW_DAYS;

  const shouldFetch = hasCoord && !outOfWindow;

  const weather = useWeather(
    shouldFetch
      ? { lat: trip.latitude as number, lng: trip.longitude as number, units: "metric" }
      : null,
    { refetchOnWindowFocus: true, staleTime: 5 * 60_000 },
  );

  const tripDays = useMemo(
    () => tripDaysWithinWindow(trip.start_date, trip.end_date, weather.data?.daily ?? []),
    [trip.start_date, trip.end_date, weather.data?.daily],
  );

  if (!hasCoord || outOfWindow) return null;

  return (
    <section
      className="rounded-2xl border p-4 flex flex-col gap-3"
      style={{ borderColor: "#1F2A24", backgroundColor: "#121814" }}
    >
      <div className="flex items-center justify-between">
        <p
          className="text-[11px] font-medium tracking-[0.2em] uppercase"
          style={{ color: "var(--season-button)" }}
        >
          Forecast · {trip.location || "Trip location"}
        </p>
        {weather.isFetching && (
          <Loader2 className="size-3.5 animate-spin text-[#8B9A8E]" />
        )}
      </div>
      {weather.isLoading ? (
        <p className="text-xs text-[#8B9A8E]">Loading forecast…</p>
      ) : weather.isError ? (
        <p className="text-xs text-[#FCA5A5]">Forecast unavailable.</p>
      ) : tripDays.length === 0 ? (
        <p className="text-xs text-[#8B9A8E]">
          Trip days fall outside the 30-day forecast window.
        </p>
      ) : (
        <ul className="grid grid-flow-col auto-cols-fr gap-2">
          {tripDays.map((d) => (
            <li
              key={d.date}
              className="rounded-xl border px-3 py-2 flex flex-col items-center gap-1"
              style={{ borderColor: "#1F2A24", backgroundColor: "#0B100D" }}
            >
              <span className="text-[10px] uppercase tracking-widest text-[#8B9A8E]">
                {formatWeekday(d.date)}
              </span>
              <span className="text-xs text-[#ECEFEA]">
                {formatMonthDay(d.date)}
              </span>
              {d.icon && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={d.icon}
                  alt={d.summary ?? ""}
                  width={40}
                  height={40}
                  loading="lazy"
                  style={{ width: 40, height: 40 }}
                />
              )}
              <span className="text-sm text-[#ECEFEA]">
                {Math.round(d.temp_max_c)}°
                <span className="text-[#8B9A8E]">
                  /{Math.round(d.temp_min_c)}°
                </span>
              </span>
              {d.summary && (
                <span className="text-[10px] text-[#8B9A8E] truncate max-w-full capitalize">
                  {d.summary}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function tripDaysWithinWindow(
  startISO: string,
  endISO: string,
  daily: WeatherDay[],
): WeatherDay[] {
  const start = parseDate(startISO);
  const end = parseDate(endISO);
  if (!start || !end) return [];
  return daily.filter((d) => {
    const t = parseDate(d.date);
    if (!t) return false;
    return t >= start && t <= end;
  });
}

function parseDate(s: string): Date {
  const iso = s.length >= 10 ? s.slice(0, 10) : s;
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, (m ?? 1) - 1, d ?? 1);
}

function daysBetween(a: Date, b: Date): number {
  const ms = b.getTime() - startOfDay(a).getTime();
  return Math.floor(ms / 86_400_000);
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function formatWeekday(iso: string): string {
  const d = parseDate(iso);
  return d.toLocaleDateString(undefined, { weekday: "short" });
}

function formatMonthDay(iso: string): string {
  const d = parseDate(iso);
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
