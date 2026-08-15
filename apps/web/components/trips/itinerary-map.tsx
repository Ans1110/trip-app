"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import L from "leaflet";
import {
  MapContainer,
  Marker,
  Polyline,
  Popup,
  TileLayer,
  useMap,
} from "react-leaflet";
import "leaflet/dist/leaflet.css";

import { useComputeRoute } from "@/hooks/location-hooks";
import type { Route } from "@/lib/apis/location-api";

export type ItineraryStop = {
  id: string;
  day: number;
  lat: number;
  lng: number;
  title: string;
  order: number;
};

const DAY_COLORS = [
  "#B5D086",
  "#79B4E6",
  "#F3B664",
  "#EC7E7E",
  "#B292E7",
  "#7BC9C7",
  "#E68FB7",
  "#C7C77B",
];

const dayColor = (d: number) => DAY_COLORS[(d - 1) % DAY_COLORS.length];

const numberedIcon = (label: string, tint: string) =>
  L.divIcon({
    className: "itinerary-pin",
    html: `<div style="width:28px;height:36px;position:relative;">
  <svg viewBox="0 0 24 32" width="28" height="36" xmlns="http://www.w3.org/2000/svg"><path fill="${tint}" stroke="#0B100D" stroke-width="1.5" d="M12 1c5 0 9 4 9 9 0 7-9 21-9 21S3 17 3 10c0-5 4-9 9-9z"/><circle cx="12" cy="10" r="6.5" fill="#0B100D"/></svg>
  <span style="position:absolute;top:1px;left:0;width:28px;text-align:center;font-size:10px;font-weight:700;color:#ECEFEA;line-height:22px;letter-spacing:-0.2px;">${escapeHtml(label)}</span>
</div>`,
    iconSize: [28, 36],
    iconAnchor: [14, 36],
    popupAnchor: [0, -34],
  });

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => {
    switch (c) {
      case "&":
        return "&amp;";
      case "<":
        return "&lt;";
      case ">":
        return "&gt;";
      case '"':
        return "&quot;";
      default:
        return "&#39;";
    }
  });
}

function FitBounds({ coords }: { coords: [number, number][] }) {
  const map = useMap();
  useEffect(() => {
    if (coords.length === 0) return;
    if (coords.length === 1) {
      map.setView(coords[0], 13);
      return;
    }
    map.fitBounds(L.latLngBounds(coords), { padding: [32, 32], maxZoom: 15 });
  }, [map, coords]);
  return null;
}

function decodePolyline(encoded?: string | null): [number, number][] {
  if (!encoded) return [];
  const points: [number, number][] = [];
  let idx = 0;
  let lat = 0;
  let lng = 0;
  while (idx < encoded.length) {
    let result = 0;
    let shift = 0;
    let byte: number;
    do {
      byte = encoded.charCodeAt(idx++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    lat += result & 1 ? ~(result >> 1) : result >> 1;
    result = 0;
    shift = 0;
    do {
      byte = encoded.charCodeAt(idx++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    lng += result & 1 ? ~(result >> 1) : result >> 1;
    points.push([lat / 1e5, lng / 1e5]);
  }
  return points;
}

type Selection = number | "all";

export default function ItineraryMap({ stops }: { stops: ItineraryStop[] }) {
  const days = useMemo(
    () => Array.from(new Set(stops.map((s) => s.day))).sort((a, b) => a - b),
    [stops],
  );

  const [selected, setSelected] = useState<Selection>(() => days[0] ?? "all");
  const effective: Selection = useMemo(() => {
    if (selected === "all") return "all";
    if (days.length === 0) return "all";
    if (!days.includes(selected)) return days[0];
    return selected;
  }, [days, selected]);

  const visible = useMemo(() => {
    const list =
      effective === "all" ? stops : stops.filter((s) => s.day === effective);
    return [...list].sort(
      (a, b) => a.day - b.day || a.order - b.order,
    );
  }, [stops, effective]);

  const coords = useMemo<[number, number][]>(
    () => visible.map((s) => [s.lat, s.lng]),
    [visible],
  );

  const compute = useComputeRoute();
  const [routes, setRoutes] = useState<Record<number, Route | null>>({});
  const stopSig = useMemo(() => sigFor(stops), [stops]);
  const lastSig = useRef(stopSig);
  useEffect(() => {
    if (lastSig.current !== stopSig) {
      lastSig.current = stopSig;
      setRoutes({});
    }
  }, [stopSig]);

  useEffect(() => {
    if (typeof effective !== "number") return;
    if (routes[effective] !== undefined) return;
    const list = stops
      .filter((s) => s.day === effective)
      .sort((a, b) => a.order - b.order);
    if (list.length < 2) return;
    const day = effective;
    compute.mutate(
      {
        origin: { lat: list[0].lat, lng: list[0].lng },
        destination: {
          lat: list[list.length - 1].lat,
          lng: list[list.length - 1].lng,
        },
        waypoints: list.slice(1, -1).map((p) => ({ lat: p.lat, lng: p.lng })),
        mode: "driving",
      },
      {
        onSuccess: (r) => setRoutes((prev) => ({ ...prev, [day]: r })),
        onError: () => setRoutes((prev) => ({ ...prev, [day]: null })),
      },
    );
    // compute.mutate is stable across renders (TanStack v5)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [effective, stopSig]);

  const activeRoute = typeof effective === "number" ? routes[effective] : null;
  const polylinePoints = useMemo(
    () => decodePolyline(activeRoute?.polyline),
    [activeRoute],
  );

  const first = visible[0] ?? stops[0];
  if (!first) return null;

  const isLoadingRoute =
    typeof effective === "number" &&
    visible.length >= 2 &&
    routes[effective] === undefined;
  const activeTint =
    typeof effective === "number" ? dayColor(effective) : "#F3B664";

  return (
    <div className="flex flex-col gap-2">
      <DayTabs days={days} selected={effective} onChange={setSelected} />
      <div
        style={{
          height: 320,
          borderRadius: 12,
          overflow: "hidden",
          border: "1px solid #1F2A24",
        }}
      >
        <MapContainer
          center={[first.lat, first.lng]}
          zoom={13}
          scrollWheelZoom={false}
          style={{ height: "100%", width: "100%" }}
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          <FitBounds
            coords={coords.concat(
              polylinePoints.length > 1 ? polylinePoints : [],
            )}
          />
          {visible.map((s, i) => {
            const label =
              effective === "all" ? `D${s.day}` : String(i + 1);
            return (
              <Marker
                key={s.id}
                position={[s.lat, s.lng]}
                icon={numberedIcon(label, dayColor(s.day))}
              >
                <Popup>
                  <strong>Day {s.day}</strong>
                  <br />
                  {s.title}
                </Popup>
              </Marker>
            );
          })}
          {polylinePoints.length > 1 && (
            <Polyline
              positions={polylinePoints}
              pathOptions={{ color: activeTint, weight: 4, opacity: 0.85 }}
            />
          )}
        </MapContainer>
      </div>
      {typeof effective === "number" && (
        <RouteSummary
          route={activeRoute}
          isLoading={isLoadingRoute}
          stopCount={visible.length}
        />
      )}
    </div>
  );
}

function DayTabs({
  days,
  selected,
  onChange,
}: {
  days: number[];
  selected: Selection;
  onChange: (s: Selection) => void;
}) {
  const btnCls = (active: boolean) =>
    `text-xs px-3 py-1 rounded-full border ${
      active
        ? "bg-white/10 text-[#ECEFEA] border-white/20"
        : "text-[#8B9A8E] border-[#1F2A24] hover:bg-white/5"
    }`;
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <button
        type="button"
        onClick={() => onChange("all")}
        className={btnCls(selected === "all")}
      >
        All
      </button>
      {days.map((d) => (
        <button
          key={d}
          type="button"
          onClick={() => onChange(d)}
          className={btnCls(selected === d)}
          style={
            selected === d
              ? { borderColor: dayColor(d), color: "#ECEFEA" }
              : undefined
          }
        >
          <span
            className="inline-block size-2 rounded-full mr-1 align-middle"
            style={{ backgroundColor: dayColor(d) }}
          />
          Day {d}
        </button>
      ))}
    </div>
  );
}

function RouteSummary({
  route,
  isLoading,
  stopCount,
}: {
  route: Route | null | undefined;
  isLoading: boolean;
  stopCount: number;
}) {
  if (isLoading) {
    return <p className="text-xs text-[#8B9A8E]">Computing route…</p>;
  }
  if (stopCount === 0) {
    return (
      <p className="text-xs text-[#6B7A6F]">
        No mapped stops for this day yet.
      </p>
    );
  }
  if (stopCount === 1) {
    return (
      <p className="text-xs text-[#6B7A6F]">
        Add another stop to see a connecting route.
      </p>
    );
  }
  if (!route) {
    return (
      <p className="text-xs text-[#FCA5A5]">
        Route unavailable for these points.
      </p>
    );
  }
  return (
    <p className="text-xs text-[#8B9A8E]">
      <span className="text-[#ECEFEA]">
        {formatDistance(route.distance_meters)}
      </span>{" "}
      · {formatDuration(route.duration_seconds)} across {stopCount} stops
    </p>
  );
}

function sigFor(stops: ItineraryStop[]): string {
  return stops
    .map((s) => `${s.day}:${s.order}:${s.lat.toFixed(5)},${s.lng.toFixed(5)}`)
    .join("|");
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
