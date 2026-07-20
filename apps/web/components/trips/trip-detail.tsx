"use client";

import { useState } from "react";

import { errorMessage, useTrip } from "@/hooks/trip-hooks";

import { TripCalendar } from "./trip-calendar";
import { TripItinerary } from "./trip-itinerary";
import { TripOverview } from "./trip-overview";
import { TripRoom } from "./trip-room";
import { TripTodos } from "./trip-todos";
import { TripVote } from "./trip-vote";

type Tab = "overview" | "itinerary" | "todos" | "calendar" | "vote" | "room";

const tabs: { value: Tab; label: string }[] = [
  { value: "overview", label: "Overview" },
  { value: "itinerary", label: "Itinerary" },
  { value: "todos", label: "Todos" },
  { value: "calendar", label: "Calendar" },
  { value: "vote", label: "Vote" },
  { value: "room", label: "Room" },
];

export function TripDetail({ tripId }: { tripId: string }) {
  const [tab, setTab] = useState<Tab>("overview");
  const query = useTrip(tripId);

  if (query.isLoading) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        Loading…
      </p>
    );
  }

  if (query.isError || !query.data) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(query.error) ?? "Trip not found."}
      </p>
    );
  }

  const trip = query.data;
  const canManage = trip.my_role === "owner" || trip.my_role === "admin";

  return (
    <div className="flex flex-col gap-6">
      <nav
        className="flex items-center gap-1 p-1 rounded-full self-start"
        style={{
          backgroundColor: "#121814",
          border: "1px solid #1F2A24",
        }}
      >
        {tabs.map((t) => {
          const active = tab === t.value;
          return (
            <button
              key={t.value}
              type="button"
              onClick={() => setTab(t.value)}
              className="season-transition px-4 py-1.5 rounded-full text-xs font-medium"
              style={{
                backgroundColor: active
                  ? "color-mix(in srgb, var(--season-button) 22%, transparent)"
                  : "transparent",
                color: active ? "#ECEFEA" : "#8B9A8E",
              }}
            >
              {t.label}
            </button>
          );
        })}
      </nav>

      {tab === "overview" && (
        <TripOverview trip={trip} canManage={canManage} />
      )}
      {tab === "itinerary" && <TripItinerary trip={trip} />}
      {tab === "todos" && <TripTodos tripId={trip.id} />}
      {tab === "calendar" && <TripCalendar tripId={trip.id} />}
      {tab === "vote" && <TripVote tripId={trip.id} />}
      {tab === "room" && <TripRoom tripId={trip.id} canManage={canManage} />}
    </div>
  );
}
