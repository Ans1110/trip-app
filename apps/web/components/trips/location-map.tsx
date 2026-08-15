"use client";

import { useEffect, useMemo, useRef } from "react";
import L from "leaflet";
import { MapContainer, Marker, Popup, TileLayer, useMap } from "react-leaflet";
import "leaflet/dist/leaflet.css";

import type { TripMemberPosition } from "@/lib/apis/location-api";
import type { RoomMember } from "@/lib/apis/trip-api";

export type LocationMapMe = {
  id: string;
  lat: number;
  lng: number;
  name: string;
  avatarUrl?: string;
};

export type FocusTarget = {
  lat: number;
  lng: number;
  key: number;
};

export type LocationMapProps = {
  me: LocationMapMe | null;
  positions: TripMemberPosition[];
  membersById: Map<string, RoomMember>;
  meIsSharing: boolean;
  focusTarget?: FocusTarget | null;
};

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

function avatarPin(opts: {
  url?: string;
  name: string;
  ring: string;
  ringDashed: boolean;
}): L.DivIcon {
  const initial = escapeHtml((opts.name || "?").slice(0, 1).toUpperCase());
  const inner = opts.url
    ? `<img src="${escapeHtml(opts.url)}" alt="" style="width:34px;height:34px;border-radius:50%;object-fit:cover;display:block;" />`
    : `<span style="width:34px;height:34px;border-radius:50%;background:#1F2A24;color:#ECEFEA;display:flex;align-items:center;justify-content:center;font:600 12px/1 system-ui,-apple-system,sans-serif;">${initial}</span>`;
  const ringStyle = opts.ringDashed ? "dashed" : "solid";
  return L.divIcon({
    className: "trip-location-pin",
    html: `<div style="position:relative;width:44px;height:52px;">
  <div style="width:44px;height:44px;border-radius:50%;padding:3px;background:#0B100D;border:2px ${ringStyle} ${opts.ring};box-sizing:border-box;box-shadow:0 4px 10px rgba(0,0,0,0.35);">${inner}</div>
  <div style="position:absolute;bottom:0;left:50%;transform:translateX(-50%);width:0;height:0;border-left:6px solid transparent;border-right:6px solid transparent;border-top:8px solid ${opts.ring};"></div>
</div>`,
    iconSize: [44, 52],
    iconAnchor: [22, 52],
    popupAnchor: [0, -50],
  });
}

function formatRelative(iso: string): string {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return iso;
  const seconds = Math.max(0, Math.floor((Date.now() - t) / 1000));
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours} h ago`;
}

function AutoCenter({
  me,
  positions,
}: {
  me: LocationMapMe | null;
  positions: TripMemberPosition[];
}) {
  const map = useMap();
  const centeredRef = useRef(false);
  useEffect(() => {
    if (centeredRef.current) return;
    if (me) {
      map.setView([me.lat, me.lng], 14);
      centeredRef.current = true;
      return;
    }
    if (positions.length > 0) {
      map.setView([positions[0].lat, positions[0].lng], 13);
      centeredRef.current = true;
    }
  }, [map, me, positions]);
  return null;
}

function ExposeMap({
  onReady,
}: {
  onReady: (m: L.Map) => void;
}) {
  const map = useMap();
  useEffect(() => {
    onReady(map);
  }, [map, onReady]);
  return null;
}

function FocusOn({ target }: { target: FocusTarget | null | undefined }) {
  const map = useMap();
  useEffect(() => {
    if (!target) return;
    map.flyTo([target.lat, target.lng], Math.max(map.getZoom(), 15), {
      duration: 0.6,
    });
  }, [map, target]);
  return null;
}

export default function LocationMap({
  me,
  positions,
  membersById,
  meIsSharing,
  focusTarget,
}: LocationMapProps) {
  const mapRef = useRef<L.Map | null>(null);

  const initialCenter = useMemo<[number, number]>(() => {
    if (me) return [me.lat, me.lng];
    if (positions[0]) return [positions[0].lat, positions[0].lng];
    return [23.6978, 120.9605];
  }, [me, positions]);
  const initialZoom = me || positions.length > 0 ? 13 : 5;

  const others = useMemo(
    () => (me ? positions.filter((p) => p.user_id !== me.id) : positions),
    [me, positions],
  );

  const jumpTo = (lat: number, lng: number) => {
    const m = mapRef.current;
    if (!m) return;
    m.flyTo([lat, lng], Math.max(m.getZoom(), 15), { duration: 0.6 });
  };

  return (
    <div
      style={{
        height: 420,
        borderRadius: 16,
        overflow: "hidden",
        border: "1px solid #1F2A24",
      }}
    >
      <MapContainer
        center={initialCenter}
        zoom={initialZoom}
        scrollWheelZoom
        style={{ height: "100%", width: "100%" }}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        <ExposeMap
          onReady={(m) => {
            mapRef.current = m;
          }}
        />
        <AutoCenter me={me} positions={positions} />
        <FocusOn target={focusTarget} />

        {me && (
          <Marker
            position={[me.lat, me.lng]}
            icon={avatarPin({
              url: me.avatarUrl,
              name: me.name,
              ring: meIsSharing ? "#B5D086" : "#8B9A8E",
              ringDashed: !meIsSharing,
            })}
            eventHandlers={{ click: () => jumpTo(me.lat, me.lng) }}
          >
            <Popup>
              <strong>{me.name}</strong> (you)
              <br />
              {meIsSharing
                ? "Sharing live with the trip."
                : "Only visible to you. Tap Share to broadcast."}
            </Popup>
          </Marker>
        )}

        {others.map((p) => {
          const member = membersById.get(p.user_id);
          const name = member?.user.name ?? "Member";
          return (
            <Marker
              key={p.user_id}
              position={[p.lat, p.lng]}
              icon={avatarPin({
                url: member?.user.avatar_url,
                name,
                ring: "#B5D086",
                ringDashed: false,
              })}
              eventHandlers={{ click: () => jumpTo(p.lat, p.lng) }}
            >
              <Popup>
                <strong>{name}</strong>
                <br />
                Shared {formatRelative(p.shared_at)}
                {p.accuracy_m
                  ? ` · ±${Math.round(p.accuracy_m)} m`
                  : ""}
                <br />
                <a
                  href={`https://www.google.com/maps?q=${p.lat},${p.lng}`}
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  Open in Maps
                </a>
              </Popup>
            </Marker>
          );
        })}
      </MapContainer>
    </div>
  );
}
