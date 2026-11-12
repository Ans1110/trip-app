"use client";

import { useEffect, useMemo, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { LocateFixed, Minus, Plus } from "lucide-react";

import { cn } from "@/lib/utils";

export type MapPin = {
  id: string;
  lat: number;
  lng: number;
  label: string;
  kind: "result" | "saved";
};

export type MapView = { lat: number; lng: number; radius: number; zoom: number };

export type MapInsets = { left: number; bottom: number };

type Props = {
  pins: MapPin[];
  selectedId?: string | null;
  hoveredId?: string | null;
  route?: { polyline?: string; origin: L.LatLngLiteral; destination: L.LatLngLiteral } | null;
  me?: L.LatLngLiteral | null;
  initialCenter: L.LatLngLiteral;
  /** Change `key` to fly to a point. */
  focus?: { lat: number; lng: number; zoom?: number; key: number } | null;
  /** Change to fit all pins (and the route) into view. */
  fitKey?: number;
  insets: MapInsets;
  locating?: boolean;
  onPinClick: (pin: MapPin) => void;
  onMapClick?: () => void;
  onViewChange: (view: MapView, byUser: boolean) => void;
  onLocate: () => void;
};

// Keeps the map controls above the fixed chat/friends buttons (56px + gaps).
const FAB_CLEARANCE = 92;

const TILE_URL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
const TILE_ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const pinIcon = (kind: MapPin["kind"], label: string) =>
  L.divIcon({
    className: "pm-pin-wrap",
    html: `<span class="pm-pin pm-${kind}" title="${escapeHtml(label)}"><span class="pm-pin-core"></span></span>`,
    iconSize: kind === "saved" ? [22, 22] : [28, 36],
    iconAnchor: kind === "saved" ? [11, 11] : [14, 34],
  });

const meIcon = L.divIcon({
  className: "pm-pin-wrap",
  html: '<span class="pm-me"><span class="pm-me-pulse"></span><span class="pm-me-dot"></span></span>',
  iconSize: [22, 22],
  iconAnchor: [11, 11],
});

const endpointIcon = (kind: "origin" | "destination") =>
  L.divIcon({
    className: "pm-pin-wrap",
    html: `<span class="pm-${kind}"></span>`,
    iconSize: kind === "origin" ? [18, 18] : [28, 36],
    iconAnchor: kind === "origin" ? [9, 9] : [14, 34],
  });

export default function PlacesMap({
  pins,
  selectedId,
  hoveredId,
  route,
  me,
  initialCenter,
  focus,
  fitKey,
  insets,
  locating,
  onPinClick,
  onMapClick,
  onViewChange,
  onLocate,
}: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<L.Map | null>(null);
  const pinLayerRef = useRef<L.LayerGroup | null>(null);
  const routeLayerRef = useRef<L.LayerGroup | null>(null);
  const meMarkerRef = useRef<L.Marker | null>(null);
  const markersRef = useRef(new Map<string, L.Marker>());
  const programmaticRef = useRef(false);
  const insetsRef = useRef(insets);
  const handlers = useRef({ onPinClick, onMapClick, onViewChange });

  useEffect(() => {
    handlers.current = { onPinClick, onMapClick, onViewChange };
    insetsRef.current = insets;
  });

  const initialCenterRef = useRef(initialCenter);
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const map = L.map(el, {
      center: initialCenterRef.current,
      zoom: 13,
      zoomControl: false,
      worldCopyJump: true,
    });
    map.attributionControl.setPrefix(false);
    L.tileLayer(TILE_URL, { attribution: TILE_ATTRIBUTION, maxZoom: 19 }).addTo(map);
    pinLayerRef.current = L.layerGroup().addTo(map);
    routeLayerRef.current = L.layerGroup().addTo(map);

    const emitView = () => {
      const c = map.getCenter();
      const b = map.getBounds();
      const radius = Math.min(50_000, Math.max(300, Math.round(c.distanceTo(b.getNorthEast()) * 0.75)));
      const byUser = !programmaticRef.current;
      programmaticRef.current = false;
      handlers.current.onViewChange({ lat: c.lat, lng: c.lng, radius, zoom: map.getZoom() }, byUser);
    };
    map.on("moveend", emitView);
    map.on("click", () => handlers.current.onMapClick?.());
    mapRef.current = map;
    emitView();

    const markers = markersRef.current;
    return () => {
      map.remove();
      mapRef.current = null;
      markers.clear();
    };
  }, []);

  useEffect(() => {
    const layer = pinLayerRef.current;
    if (!layer) return;
    layer.clearLayers();
    markersRef.current.clear();
    for (const pin of pins) {
      const marker = L.marker([pin.lat, pin.lng], {
        icon: pinIcon(pin.kind, pin.label),
        keyboard: true,
        title: pin.label,
        alt: pin.label,
        riseOnHover: true,
      });
      marker.on("click", (e) => {
        L.DomEvent.stopPropagation(e);
        handlers.current.onPinClick(pin);
      });
      layer.addLayer(marker);
      markersRef.current.set(pin.id, marker);
    }
  }, [pins]);

  useEffect(() => {
    for (const [id, marker] of markersRef.current) {
      const el = marker.getElement()?.querySelector(".pm-pin");
      const selected = id === selectedId;
      const hovered = id === hoveredId;
      el?.classList.toggle("is-selected", selected);
      el?.classList.toggle("is-hovered", hovered && !selected);
      marker.setZIndexOffset(selected ? 1000 : hovered ? 500 : 0);
    }
  }, [selectedId, hoveredId, pins]);

  const routePoints = useMemo(() => decodePolyline(route?.polyline), [route?.polyline]);

  useEffect(() => {
    const layer = routeLayerRef.current;
    if (!layer) return;
    layer.clearLayers();
    if (!route) return;
    if (routePoints.length > 1) {
      L.polyline(routePoints, { color: "#ffffff", weight: 10, opacity: 0.95, lineCap: "round", lineJoin: "round" }).addTo(layer);
      L.polyline(routePoints, { className: "pm-route", weight: 5, lineCap: "round", lineJoin: "round" }).addTo(layer);
    }
    L.marker(route.origin, { icon: endpointIcon("origin"), interactive: false }).addTo(layer);
    L.marker(route.destination, { icon: endpointIcon("destination"), interactive: false, zIndexOffset: 1200 }).addTo(layer);
  }, [route, routePoints]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    if (!me) {
      meMarkerRef.current?.remove();
      meMarkerRef.current = null;
      return;
    }
    if (meMarkerRef.current) meMarkerRef.current.setLatLng(me);
    else meMarkerRef.current = L.marker(me, { icon: meIcon, interactive: false, zIndexOffset: 800 }).addTo(map);
  }, [me]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !focus) return;
    programmaticRef.current = true;
    const zoom = focus.zoom ?? Math.max(map.getZoom(), 15);
    const target = map.project([focus.lat, focus.lng], zoom).subtract([
      (insetsRef.current.left) / 2,
      -(insetsRef.current.bottom) / 2,
    ]);
    map.flyTo(map.unproject(target, zoom), zoom, { duration: prefersReducedMotion() ? 0 : 0.6 });
  }, [focus]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !fitKey) return;
    const coords: L.LatLngExpression[] = [
      ...pins.map((p) => [p.lat, p.lng] as [number, number]),
      ...routePoints,
    ];
    if (coords.length === 0) return;
    programmaticRef.current = true;
    const { left, bottom } = insetsRef.current;
    if (coords.length === 1) {
      map.setView(coords[0], Math.max(map.getZoom(), 15));
      return;
    }
    map.fitBounds(L.latLngBounds(coords), {
      paddingTopLeft: [left + 32, 96],
      paddingBottomRight: [72, bottom + 32],
      maxZoom: 16,
      animate: !prefersReducedMotion(),
    });
    // fitKey alone drives fitting; pins/route are read at that moment.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fitKey]);

  return (
    <div className="relative size-full">
      <div ref={containerRef} className="places-map size-full" />
      <div
        className="absolute right-3 z-[500] flex flex-col gap-2 transition-[bottom] duration-300"
        style={{ bottom: insets.bottom + FAB_CLEARANCE }}
      >
        <MapButton label="Show my location" onClick={onLocate} busy={locating}>
          <LocateFixed className="size-[18px]" />
        </MapButton>
        <div className="flex flex-col rounded-full bg-card shadow-soft overflow-hidden">
          <button
            type="button"
            aria-label="Zoom in"
            onClick={() => mapRef.current?.zoomIn()}
            className="grid place-items-center size-10 text-foreground hover:bg-muted outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
          >
            <Plus className="size-[18px]" />
          </button>
          <span className="h-px mx-2 bg-border" aria-hidden />
          <button
            type="button"
            aria-label="Zoom out"
            onClick={() => mapRef.current?.zoomOut()}
            className="grid place-items-center size-10 text-foreground hover:bg-muted outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
          >
            <Minus className="size-[18px]" />
          </button>
        </div>
      </div>
    </div>
  );
}

function MapButton({
  label,
  onClick,
  busy,
  children,
}: {
  label: string;
  onClick: () => void;
  busy?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      aria-busy={busy || undefined}
      className={cn(
        "grid place-items-center size-10 rounded-full bg-card shadow-soft text-sky-ink hover:bg-muted outline-none focus-visible:ring-2 focus-visible:ring-ring",
        busy && "animate-pulse",
      )}
    >
      {children}
    </button>
  );
}

function prefersReducedMotion() {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function escapeHtml(s: string) {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!);
}

function decodePolyline(encoded?: string): [number, number][] {
  if (!encoded) return [];
  const points: [number, number][] = [];
  let index = 0;
  let lat = 0;
  let lng = 0;
  while (index < encoded.length) {
    for (const axis of [0, 1]) {
      let result = 0;
      let shift = 0;
      let byte: number;
      do {
        byte = encoded.charCodeAt(index++) - 63;
        result |= (byte & 0x1f) << shift;
        shift += 5;
      } while (byte >= 0x20);
      const delta = result & 1 ? ~(result >> 1) : result >> 1;
      if (axis === 0) lat += delta;
      else lng += delta;
    }
    points.push([lat / 1e5, lng / 1e5]);
  }
  return points;
}
