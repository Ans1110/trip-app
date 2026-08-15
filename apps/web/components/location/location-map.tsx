"use client";

import { useEffect, useMemo, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

export type MapPoint = {
  id: string;
  lat: number;
  lng: number;
  label?: string;
  kind: "saved" | "result" | "selected" | "origin" | "destination";
};

type Props = {
  points: MapPoint[];
  polyline?: string;
  center?: { lat: number; lng: number };
  zoom?: number;
  onMapClick?: (lat: number, lng: number) => void;
  onMarkerClick?: (point: MapPoint) => void;
};

const COLORS: Record<MapPoint["kind"], string> = {
  saved: "#EF6E6E",
  result: "#79B4E6",
  selected: "#F3B664",
  origin: "#79B4E6",
  destination: "#F3B664",
};

const STAR_POINTS =
  "12,6.5 12.82,8.87 15.33,8.92 13.33,10.43 14.06,12.83 12,11.4 9.94,12.83 10.67,10.43 8.67,8.92 11.18,8.87";

const iconFor = (kind: MapPoint["kind"]) =>
  L.icon({
    iconUrl:
      "data:image/svg+xml;utf8," +
      encodeURIComponent(
        `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 32" width="24" height="32"><path fill="${COLORS[kind]}" stroke="#0B100D" stroke-width="1.5" d="M12 1c5 0 9 4 9 9 0 7-9 21-9 21S3 17 3 10c0-5 4-9 9-9z"/><polygon fill="#0B100D" points="${STAR_POINTS}"/></svg>`,
      ),
    iconSize: [24, 32],
    iconAnchor: [12, 32],
    popupAnchor: [0, -30],
  });

export default function LocationMap({
  points,
  polyline,
  center,
  zoom,
  onMapClick,
  onMarkerClick,
}: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markersLayerRef = useRef<L.LayerGroup | null>(null);
  const polylineRef = useRef<L.Polyline | null>(null);
  const onMapClickRef = useRef(onMapClick);
  const onMarkerClickRef = useRef(onMarkerClick);

  useEffect(() => {
    onMapClickRef.current = onMapClick;
  }, [onMapClick]);
  useEffect(() => {
    onMarkerClickRef.current = onMarkerClick;
  }, [onMarkerClick]);

  const polylinePoints = useMemo(() => decodePolyline(polyline), [polyline]);

  const initialCenter = useMemo<[number, number]>(() => {
    if (center) return [center.lat, center.lng];
    if (points[0]) return [points[0].lat, points[0].lng];
    return [25.033, 121.5654];
  }, [center, points]);
  const initialZoom = zoom ?? 12;
  // Read once at mount; subsequent center/zoom prop changes are handled by the
  // center-tracking effect below.
  const initialCenterRef = useRef(initialCenter);
  const initialZoomRef = useRef(initialZoom);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const map = L.map(el, {
      center: initialCenterRef.current,
      zoom: initialZoomRef.current,
      scrollWheelZoom: true,
    });
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    }).addTo(map);
    map.on("click", (e: L.LeafletMouseEvent) => {
      onMapClickRef.current?.(e.latlng.lat, e.latlng.lng);
    });
    markersLayerRef.current = L.layerGroup().addTo(map);
    mapRef.current = map;
    return () => {
      map.remove();
      mapRef.current = null;
      markersLayerRef.current = null;
      polylineRef.current = null;
    };
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !center) return;
    map.setView([center.lat, center.lng], zoom ?? map.getZoom());
  }, [center, zoom]);

  useEffect(() => {
    const map = mapRef.current;
    const layer = markersLayerRef.current;
    if (!map || !layer) return;
    layer.clearLayers();
    for (const p of points) {
      const marker = L.marker([p.lat, p.lng], { icon: iconFor(p.kind) });
      if (p.label) marker.bindPopup(p.label);
      marker.on("click", () => onMarkerClickRef.current?.(p));
      layer.addLayer(marker);
    }
  }, [points]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    if (polylineRef.current) {
      map.removeLayer(polylineRef.current);
      polylineRef.current = null;
    }
    if (polylinePoints.length > 1) {
      polylineRef.current = L.polyline(polylinePoints, {
        color: "#F3B664",
        weight: 4,
        opacity: 0.85,
      }).addTo(map);
    }
  }, [polylinePoints]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    const coords: [number, number][] = points
      .map((p) => [p.lat, p.lng] as [number, number])
      .concat(polylinePoints);
    if (coords.length === 0) return;
    if (coords.length === 1) {
      map.setView(coords[0], Math.max(map.getZoom(), 13));
      return;
    }
    const bounds = L.latLngBounds(coords);
    map.fitBounds(bounds, { padding: [32, 32], maxZoom: 15 });
  }, [points, polylinePoints]);

  return (
    <div ref={containerRef} style={{ height: "100%", width: "100%" }} />
  );
}

function decodePolyline(encoded?: string): [number, number][] {
  if (!encoded) return [];
  const points: [number, number][] = [];
  let index = 0;
  let lat = 0;
  let lng = 0;
  while (index < encoded.length) {
    let result = 0;
    let shift = 0;
    let byte: number;
    do {
      byte = encoded.charCodeAt(index++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    const dlat = result & 1 ? ~(result >> 1) : result >> 1;
    lat += dlat;

    result = 0;
    shift = 0;
    do {
      byte = encoded.charCodeAt(index++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    const dlng = result & 1 ? ~(result >> 1) : result >> 1;
    lng += dlng;

    points.push([lat / 1e5, lng / 1e5]);
  }
  return points;
}
