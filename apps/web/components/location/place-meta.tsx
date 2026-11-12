"use client";

import { useState } from "react";
import { MapPin, Star } from "lucide-react";

import type { PlaceHours } from "@/lib/apis/location-api";
import { openStatus } from "@/lib/place-hours";
import { cn } from "@/lib/utils";

const GENERIC_TYPES = new Set(["point_of_interest", "establishment", "premise", "political", "geocode"]);

export function typeLabel(types?: string[]): string | null {
  const t = types?.find((x) => !GENERIC_TYPES.has(x));
  if (!t) return null;
  const s = t.replace(/_/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export const photoSrc = (name: string, width: number) =>
  `/api/location/photo?name=${encodeURIComponent(name)}&w=${width}`;

const FALLBACK_TINTS = [
  "linear-gradient(135deg, var(--mint) 0%, var(--cyan-soft) 100%)",
  "linear-gradient(135deg, var(--pink) 0%, var(--sky-soft) 100%)",
  "linear-gradient(135deg, var(--sky) 0%, var(--mint-soft) 100%)",
];

export function PlacePhoto({
  name,
  width,
  alt,
  seed,
  className,
}: {
  name?: string;
  width: number;
  alt: string;
  seed: string;
  className?: string;
}) {
  const [failed, setFailed] = useState(false);
  const tint = FALLBACK_TINTS[hash(seed) % FALLBACK_TINTS.length];
  if (!name || failed) {
    return (
      <div className={cn("grid place-items-center", className)} style={{ background: tint }} aria-hidden>
        <MapPin className="size-5 text-white/90" />
      </div>
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={photoSrc(name, width)}
      alt={alt}
      loading="lazy"
      decoding="async"
      onError={() => setFailed(true)}
      className={cn("object-cover", className)}
      style={{ background: tint }}
    />
  );
}

export function RatingLine({
  rating,
  count,
  priceLevel,
  types,
  className,
}: {
  rating?: number;
  count?: number;
  priceLevel?: number;
  types?: string[];
  className?: string;
}) {
  const kind = typeLabel(types);
  const price = priceLevel ? "$".repeat(priceLevel) : null;
  const parts: React.ReactNode[] = [];
  if (rating) {
    parts.push(
      <span key="r" className="inline-flex items-center gap-1">
        <span className="font-medium text-foreground tabular-nums">{rating.toFixed(1)}</span>
        <Star className="size-3.5 fill-[var(--warning)] text-warning-ink" aria-hidden />
        {count ? <span className="tabular-nums">({formatCount(count)})</span> : null}
      </span>,
    );
  }
  if (kind) parts.push(<span key="k">{kind}</span>);
  if (price) parts.push(<span key="p">{price}</span>);
  if (parts.length === 0) return null;
  return (
    <p className={cn("flex flex-wrap items-center gap-x-1.5 text-sm text-muted-foreground", className)}>
      {parts.map((p, i) => (
        <span key={i} className="inline-flex items-center gap-1.5">
          {i > 0 && <span aria-hidden>·</span>}
          {p}
        </span>
      ))}
    </p>
  );
}

export function OpenBadge({ hours, className }: { hours?: PlaceHours; className?: string }) {
  const s = openStatus(hours);
  if (!s) return null;
  const tone =
    s.tone === "open" ? "text-mint-ink" : s.tone === "closing" ? "text-warning-ink" : "text-destructive";
  return (
    <p className={cn("text-sm", className)}>
      <span className={cn("font-medium", tone)}>{s.label}</span>
      {s.detail && <span className="text-muted-foreground"> · {s.detail}</span>}
    </p>
  );
}

function formatCount(n: number) {
  return n >= 1000 ? `${(n / 1000).toFixed(n >= 10_000 ? 0 : 1)}k` : String(n);
}

function hash(s: string) {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
  return Math.abs(h);
}
