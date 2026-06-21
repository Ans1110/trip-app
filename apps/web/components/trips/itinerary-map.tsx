"use client";

import { useEffect } from "react";
import L from "leaflet";
import { MapContainer, Marker, TileLayer, useMap } from "react-leaflet";
import "leaflet/dist/leaflet.css";

type MapMarker = {
  id: string;
  lat: number;
  lng: number;
  label: string;
};

const icon = L.icon({
  iconUrl:
    "data:image/svg+xml;utf8," +
    encodeURIComponent(
      `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 32" width="24" height="32"><path fill="%23B5D086" stroke="%230B100D" stroke-width="1.5" d="M12 1c5 0 9 4 9 9 0 7-9 21-9 21S3 17 3 10c0-5 4-9 9-9z"/><circle cx="12" cy="10" r="3" fill="%230B100D"/></svg>`,
    ),
  iconSize: [24, 32],
  iconAnchor: [12, 32],
});

function FitBounds({ markers }: { markers: MapMarker[] }) {
  const map = useMap();
  useEffect(() => {
    if (markers.length === 0) return;
    if (markers.length === 1) {
      map.setView([markers[0].lat, markers[0].lng], 13);
      return;
    }
    const bounds = L.latLngBounds(markers.map((m) => [m.lat, m.lng]));
    map.fitBounds(bounds, { padding: [24, 24] });
  }, [map, markers]);
  return null;
}

export default function ItineraryMap({ markers }: { markers: MapMarker[] }) {
  if (markers.length === 0) return null;
  const first = markers[0];
  return (
    <MapContainer
      center={[first.lat, first.lng]}
      zoom={13}
      scrollWheelZoom={false}
      style={{ height: 240, width: "100%", borderRadius: 12 }}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      <FitBounds markers={markers} />
      {markers.map((m) => (
        <Marker key={m.id} position={[m.lat, m.lng]} icon={icon} title={m.label} />
      ))}
    </MapContainer>
  );
}
