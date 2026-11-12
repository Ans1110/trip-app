"use client";

import { Loader2, SearchX } from "lucide-react";

import { cn } from "@/lib/utils";

import type { PlaceRef } from "./place-details";
import { OpenBadge, PlacePhoto, RatingLine } from "./place-meta";

type Props = {
  title: string;
  places: PlaceRef[];
  loading: boolean;
  error: string | null;
  emptyText: string;
  hoveredKey: string | null;
  onHover: (key: string | null) => void;
  onSelect: (place: PlaceRef) => void;
};

export function PlaceResults({ title, places, loading, error, emptyText, hoveredKey, onHover, onSelect }: Props) {
  return (
    <section aria-label={title} className="flex flex-col">
      <h2 className="text-base text-foreground px-1 pb-2">{title}</h2>
      {loading ? (
        <ul className="flex flex-col gap-1" aria-busy>
          {Array.from({ length: 4 }, (_, i) => (
            <li key={i} className="flex gap-3 p-2">
              <div className="size-[72px] rounded-2xl bg-muted animate-pulse shrink-0" />
              <div className="flex-1 flex flex-col gap-2 pt-1">
                <div className="h-4 w-2/3 rounded-full bg-muted animate-pulse" />
                <div className="h-3 w-1/2 rounded-full bg-muted animate-pulse" />
                <div className="h-3 w-1/3 rounded-full bg-muted animate-pulse" />
              </div>
            </li>
          ))}
          <li className="sr-only">
            <Loader2 /> Loading
          </li>
        </ul>
      ) : error ? (
        <p className="rounded-2xl bg-pink-soft px-4 py-3 text-sm text-destructive">{error}</p>
      ) : places.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-10 text-center">
          <span className="grid place-items-center size-12 rounded-full bg-sky-soft text-sky-ink">
            <SearchX className="size-5" />
          </span>
          <p className="text-sm text-muted-foreground max-w-[16rem]">{emptyText}</p>
        </div>
      ) : (
        <ul className="flex flex-col">
          {places.map((p) => (
            <li key={p.key}>
              <button
                type="button"
                onClick={() => onSelect(p)}
                onMouseEnter={() => onHover(p.key)}
                onMouseLeave={() => onHover(null)}
                onFocus={() => onHover(p.key)}
                onBlur={() => onHover(null)}
                className={cn(
                  "w-full flex gap-3 p-2 rounded-2xl text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  hoveredKey === p.key ? "bg-muted" : "hover:bg-muted",
                )}
              >
                <PlacePhoto
                  name={p.photo}
                  width={160}
                  alt=""
                  seed={p.key}
                  className="size-[72px] rounded-2xl shrink-0"
                />
                <span className="min-w-0 flex-1 pt-0.5">
                  <span className="block text-[15px] font-medium text-foreground truncate">{p.name}</span>
                  <RatingLine
                    className="text-[13px] mt-0.5"
                    rating={p.rating}
                    count={p.user_ratings_total}
                    priceLevel={p.price_level}
                    types={p.types}
                  />
                  <OpenBadge hours={p.hours} className="text-[13px]" />
                  {!p.hours && p.address && (
                    <span className="block text-[13px] text-muted-foreground truncate">{p.address}</span>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
