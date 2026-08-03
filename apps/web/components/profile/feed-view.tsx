"use client";

import Link from "next/link";
import { useState } from "react";
import { ChevronRight, Loader2, MapPin } from "lucide-react";

import { errorMessage, useFeed } from "@/hooks/profile-hooks";

export function FeedView() {
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const { data, isLoading, error } = useFeed({ cursor });

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm" style={{ color: "#8B9A8E" }}>
        <Loader2 className="size-4 animate-spin" />
        Loading feed
      </div>
    );
  }
  if (error || !data) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Failed to load feed"}
      </p>
    );
  }
  if (data.items.length === 0) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        Follow travelers to see their published trips here.
      </p>
    );
  }

  return (
    <div>
      <ul className="flex flex-col gap-3">
        {data.items.map((item) => {
          const initial =
            (item.actor.username || item.actor.name || "U").trim()[0]!.toUpperCase();
          return (
            <li
              key={item.id}
              className="flex items-center gap-4 p-4 rounded-xl border"
              style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
            >
              <Link
                href={`/profile/${item.actor.username}`}
                className="size-10 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
                style={{
                  backgroundColor: item.actor.avatar_url
                    ? "transparent"
                    : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
                  color: "#ECEFEA",
                }}
              >
                {item.actor.avatar_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={item.actor.avatar_url}
                    alt={item.actor.name}
                    className="size-full object-cover"
                  />
                ) : (
                  <span aria-hidden>{initial}</span>
                )}
              </Link>
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
                  <Link
                    href={`/profile/${item.actor.username}`}
                    className="font-semibold hover:underline"
                  >
                    {item.actor.name || `@${item.actor.username}`}
                  </Link>{" "}
                  <span style={{ color: "#8B9A8E" }}>
                    {describeEvent(item.event_type)}
                  </span>
                </p>
                <p
                  className="text-[11px] mt-0.5"
                  style={{ color: "#6B7A6F" }}
                >
                  {formatWhen(item.published_at)}
                </p>
              </div>
              {item.trip_id && (
                <Link
                  href={`/trips/${item.trip_id}`}
                  className="inline-flex items-center gap-1 text-xs px-3 py-1.5 rounded-full border hover:bg-white/5"
                  style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
                >
                  <MapPin className="size-3.5" />
                  View trip
                  <ChevronRight className="size-3.5" />
                </Link>
              )}
            </li>
          );
        })}
      </ul>

      {data.next_cursor && (
        <div className="mt-6 flex justify-center">
          <button
            type="button"
            onClick={() => setCursor(data.next_cursor)}
            className="px-4 py-2 text-sm rounded-full border hover:bg-white/5"
            style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}

function describeEvent(eventType: string): string {
  switch (eventType) {
    case "TRIP_PUBLISHED":
      return "published a trip";
    default:
      return eventType.replace(/_/g, " ").toLowerCase();
  }
}

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
