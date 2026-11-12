"use client";

import { useEffect, useRef, useState } from "react";
import {
  Backpack,
  CalendarDays,
  ChevronDown,
  DoorOpen,
  Images,
  LayoutDashboard,
  ListChecks,
  MapPin,
  Route,
  Vote,
  Wallet,
  X,
  type LucideIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";

import { errorMessage, useTrip } from "@/hooks/trip-hooks";

import { AlbumPanel } from "./album/album-panel";
import { TripCalendar } from "./trip-calendar";
import { TripFinance } from "./trip-finance";
import { TripItinerary } from "./trip-itinerary";
import { TripLocation } from "./trip-location";
import { TripOverview } from "./trip-overview";
import { TripPacking } from "./trip-packing";
import { TripRoom } from "./trip-room";
import { TripTodos } from "./trip-todos";
import { TripVote } from "./trip-vote";

type Tab =
  | "overview"
  | "itinerary"
  | "todos"
  | "calendar"
  | "vote"
  | "packing"
  | "finance"
  | "album"
  | "location"
  | "room";

const tabs: { value: Tab; label: string; icon: LucideIcon }[] = [
  { value: "overview", label: "Overview", icon: LayoutDashboard },
  { value: "itinerary", label: "Itinerary", icon: Route },
  { value: "todos", label: "Todos", icon: ListChecks },
  { value: "calendar", label: "Calendar", icon: CalendarDays },
  { value: "vote", label: "Vote", icon: Vote },
  { value: "packing", label: "Packing", icon: Backpack },
  { value: "finance", label: "Finance", icon: Wallet },
  { value: "album", label: "Album", icon: Images },
  { value: "location", label: "Location", icon: MapPin },
  { value: "room", label: "Room", icon: DoorOpen },
];

const MOBILE_PRIMARY: Tab[] = ["overview", "itinerary", "todos", "album"];

export function TripDetail({ tripId }: { tripId: string }) {
  const [tab, setTab] = useState<Tab>("overview");
  const query = useTrip(tripId);

  if (query.isLoading) {
    return (
      <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
        Loading…
      </p>
    );
  }

  if (query.isError || !query.data) {
    return (
      <p className="text-sm" style={{ color: "var(--destructive)" }}>
        {errorMessage(query.error) ?? "Trip not found."}
      </p>
    );
  }

  const trip = query.data;
  const canManage = trip.my_role === "owner" || trip.my_role === "admin";

  return (
    <div className="flex flex-col gap-6">
      <TripTabs active={tab} onChange={setTab} />

      {tab === "overview" && <TripOverview trip={trip} canManage={canManage} />}
      {tab === "itinerary" && <TripItinerary trip={trip} />}
      {tab === "todos" && <TripTodos tripId={trip.id} />}
      {tab === "calendar" && <TripCalendar tripId={trip.id} />}
      {tab === "vote" && <TripVote tripId={trip.id} />}
      {tab === "packing" && <TripPacking tripId={trip.id} />}
      {tab === "finance" && <TripFinance tripId={trip.id} />}
      {tab === "album" && <AlbumPanel tripId={trip.id} />}
      {tab === "location" && <TripLocation tripId={trip.id} />}
      {tab === "room" && <TripRoom tripId={trip.id} canManage={canManage} />}
    </div>
  );
}

function TripTabs({ active, onChange }: { active: Tab; onChange: (t: Tab) => void }) {
  const [moreOpen, setMoreOpen] = useState(false);
  const activeInMore = !MOBILE_PRIMARY.includes(active);
  const activeMeta = tabs.find((t) => t.value === active);

  return (
    <>
      {/* Desktop: ten equal columns so the bar is symmetric at any width; labels stack under icons until xl. */}
      <nav aria-label="Trip sections" className="hidden md:block">
        <div className="grid grid-cols-10 gap-1 p-1.5 rounded-[22px] xl:rounded-full bg-card border border-border shadow-soft">
          {tabs.map((t) => (
            <TabPill key={t.value} tab={t} active={active === t.value} onClick={() => onChange(t.value)} />
          ))}
        </div>
      </nav>

      {/* Mobile: four primary tabs plus More. */}
      <nav aria-label="Trip sections" className="md:hidden">
        <div className="grid grid-cols-5 gap-1 p-1 rounded-full bg-card border border-border shadow-soft">
          {MOBILE_PRIMARY.map((value) => {
            const t = tabs.find((x) => x.value === value)!;
            return (
              <TabPill key={value} tab={t} active={active === value} onClick={() => onChange(value)} compact />
            );
          })}
          <button
            type="button"
            onClick={() => setMoreOpen(true)}
            aria-haspopup="dialog"
            aria-expanded={moreOpen}
            className={cn(
              "flex flex-col items-center justify-center gap-0.5 h-12 rounded-full text-[11px] min-w-0 outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors",
              activeInMore ? "bg-mint-soft text-mint-ink font-semibold" : "text-muted-foreground",
            )}
          >
            {activeInMore && activeMeta ? (
              <activeMeta.icon className="size-[18px]" aria-hidden />
            ) : (
              <ChevronDown className="size-[18px]" aria-hidden />
            )}
            <span className="truncate max-w-full px-1">{activeInMore && activeMeta ? activeMeta.label : "More"}</span>
          </button>
        </div>
      </nav>

      {moreOpen && (
        <MoreSheet
          active={active}
          onPick={(t) => {
            onChange(t);
            setMoreOpen(false);
          }}
          onClose={() => setMoreOpen(false)}
        />
      )}
    </>
  );
}

function TabPill({
  tab,
  active,
  onClick,
  compact,
}: {
  tab: (typeof tabs)[number];
  active: boolean;
  onClick: () => void;
  compact?: boolean;
}) {
  const Icon = tab.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={cn(
        "rounded-full font-medium outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors shrink-0",
        compact
          ? "flex flex-col items-center justify-center gap-0.5 h-12 text-[11px] min-w-0"
          : "min-w-0 flex flex-col xl:flex-row items-center justify-center gap-1 xl:gap-1.5 h-14 xl:h-10 px-1 text-xs",
        active ? "bg-mint-soft text-mint-ink" : "text-muted-foreground hover:text-foreground hover:bg-muted",
      )}
    >
      <Icon className={compact ? "size-[18px]" : "size-[18px] xl:size-4 shrink-0"} aria-hidden />
      <span className="truncate max-w-full">{tab.label}</span>
    </button>
  );
}

function MoreSheet({
  active,
  onPick,
  onClose,
}: {
  active: Tab;
  onPick: (t: Tab) => void;
  onClose: () => void;
}) {
  const closeRef = useRef<HTMLButtonElement | null>(null);
  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("keydown", onKey);
      prev?.focus?.();
    };
  }, [onClose]);

  const rest = tabs.filter((t) => !MOBILE_PRIMARY.includes(t.value));

  return (
    <div className="fixed inset-0 z-50 md:hidden" role="dialog" aria-modal="true" aria-labelledby="trip-more-title">
      <button
        type="button"
        aria-label="Close"
        tabIndex={-1}
        onClick={onClose}
        className="trip-more-backdrop absolute inset-0"
      />
      <div className="trip-more-sheet absolute inset-x-0 bottom-0 rounded-t-[28px] bg-card shadow-soft-lg px-4 pt-3 pb-[calc(1.25rem+env(safe-area-inset-bottom))]">
        <div className="mx-auto mb-3 h-1.5 w-10 rounded-full bg-border" aria-hidden />
        <div className="flex items-center justify-between mb-3 px-1">
          <h2 id="trip-more-title" className="text-base text-foreground">
            More sections
          </h2>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="grid place-items-center size-9 rounded-full text-muted-foreground hover:bg-muted outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <X className="size-4" />
          </button>
        </div>
        <ul className="grid grid-cols-3 gap-2">
          {rest.map(({ value, label, icon: Icon }) => {
            const on = value === active;
            return (
              <li key={value}>
                <button
                  type="button"
                  onClick={() => onPick(value)}
                  aria-current={on ? "page" : undefined}
                  className={cn(
                    "w-full flex flex-col items-center gap-1.5 rounded-2xl py-3.5 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors",
                    on ? "bg-mint-soft text-mint-ink font-semibold" : "bg-surface-sunken text-foreground active:bg-muted",
                  )}
                >
                  <Icon className="size-5" aria-hidden />
                  {label}
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
