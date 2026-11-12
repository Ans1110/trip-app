"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import dynamic from "next/dynamic";
import {
  BedDouble,
  Bookmark,
  Coffee,
  Landmark,
  RefreshCw,
  ShoppingBag,
  TrainFront,
  Trees,
  Utensils,
} from "lucide-react";

import {
  errorMessage,
  useComputeRoute,
  useCreateSavedPlace,
  useDeleteSavedPlace,
  useNearbyPlaces,
  useSavedPlaces,
  useSearchPlaces,
} from "@/hooks/location-hooks";
import type { PlaceSummary, Route, SavedPlace, TravelMode } from "@/lib/apis/location-api";
import { cn } from "@/lib/utils";

import { DirectionsPanel, type Endpoint } from "./directions-panel";
import { PlaceDetailsPanel, type PlaceRef } from "./place-details";
import { PlaceResults } from "./place-results";
import { PlaceSearchBox, type SearchBias } from "./place-search-box";
import type { MapInsets, MapPin, MapView } from "./places-map";

const PlacesMap = dynamic(() => import("./places-map"), {
  ssr: false,
  loading: () => <div className="size-full bg-surface-sunken" />,
});

const DEFAULT_CENTER = { lat: 25.033, lng: 121.5654 };
const PANEL_WIDTH = 400;
const PANEL_GAP = 24;
const WEATHER_CATEGORY = "weather";

const CATEGORIES = [
  { type: "restaurant", label: "Restaurants", icon: Utensils },
  { type: "cafe", label: "Cafes", icon: Coffee },
  { type: "tourist_attraction", label: "Attractions", icon: Landmark },
  { type: "lodging", label: "Hotels", icon: BedDouble },
  { type: "park", label: "Parks", icon: Trees },
  { type: "shopping_mall", label: "Shopping", icon: ShoppingBag },
  { type: "transit_station", label: "Transit", icon: TrainFront },
] as const;

type Area = { lat: number; lng: number; radius: number };
type Source =
  | ({ type: "search"; q: string } & Area)
  | ({ type: "nearby"; category: string; label: string } & Area)
  | { type: "saved" };
type View = "home" | "results" | "details" | "directions";
type Snap = "peek" | "half" | "full";

const fromSummary = (p: PlaceSummary): PlaceRef => ({
  key: p.place_id,
  place_id: p.place_id,
  name: p.name,
  address: p.address,
  lat: p.location.lat,
  lng: p.location.lng,
  rating: p.rating,
  user_ratings_total: p.user_ratings_total,
  types: p.types,
  price_level: p.price_level,
  photo: p.photo,
  hours: p.hours,
});

const fromSaved = (s: SavedPlace): PlaceRef => ({
  key: s.place_id || `saved:${s.id}`,
  place_id: s.place_id || undefined,
  name: s.name,
  address: s.address,
  lat: s.lat,
  lng: s.lng,
});

export function PlacesView() {
  const [view, setView] = useState<View>("home");
  const [source, setSource] = useState<Source | null>(null);
  const [selected, setSelected] = useState<PlaceRef | null>(null);
  const [hoveredKey, setHoveredKey] = useState<string | null>(null);
  const [mapView, setMapView] = useState<MapView | null>(null);
  const [movedSinceSearch, setMovedSinceSearch] = useState(false);
  const [focus, setFocus] = useState<{ lat: number; lng: number; zoom?: number; key: number } | null>(null);
  const [fitKey, setFitKey] = useState(0);
  const [me, setMe] = useState<{ lat: number; lng: number } | null>(null);
  const [locating, setLocating] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [from, setFrom] = useState<Endpoint | null>(null);
  const [to, setTo] = useState<Endpoint | null>(null);
  const [mode, setMode] = useState<TravelMode>("driving");
  const [route, setRoute] = useState<Route | null>(null);
  const [routeError, setRouteError] = useState<string | null>(null);
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const [snap, setSnap] = useState<Snap>("half");
  const shellRef = useRef<HTMLDivElement | null>(null);
  const shellHeight = useElementHeight(shellRef);

  const bias: SearchBias | null = mapView ? { lat: mapView.lat, lng: mapView.lng, radius: mapView.radius } : null;

  // ---- data ----
  const savedQuery = useSavedPlaces();
  const savedList = useMemo(
    () => (savedQuery.data ?? []).filter((p) => p.category !== WEATHER_CATEGORY),
    [savedQuery.data],
  );
  const createSaved = useCreateSavedPlace();
  const deleteSaved = useDeleteSavedPlace();

  const search = useSearchPlaces(
    source?.type === "search"
      ? { q: source.q, lat: source.lat, lng: source.lng, radius: source.radius, limit: 20 }
      : { q: "" },
    { staleTime: 5 * 60_000 },
  );
  const nearby = useNearbyPlaces(
    source?.type === "nearby"
      ? { lat: source.lat, lng: source.lng, radius: source.radius, type: source.category, limit: 20 }
      : null,
    { staleTime: 5 * 60_000 },
  );

  const results: PlaceRef[] = useMemo(() => {
    if (source?.type === "search") return (search.data ?? []).map(fromSummary);
    if (source?.type === "nearby") return (nearby.data ?? []).map(fromSummary);
    if (source?.type === "saved") return savedList.map(fromSaved);
    return [];
  }, [source, search.data, nearby.data, savedList]);

  const resultsLoading =
    source?.type === "search" ? search.isLoading : source?.type === "nearby" ? nearby.isLoading : savedQuery.isLoading;
  const resultsErrorObj =
    source?.type === "search" ? search.error : source?.type === "nearby" ? nearby.error : savedQuery.error;
  const resultsError = resultsErrorObj && !isNoResult(resultsErrorObj) ? errorMessage(resultsErrorObj) : null;

  const savedMatch = useCallback(
    (p: PlaceRef | null) =>
      p
        ? savedList.find((s) => (p.place_id && s.place_id === p.place_id) || `saved:${s.id}` === p.key)
        : undefined,
    [savedList],
  );

  // ---- map pins ----
  const pins: MapPin[] = useMemo(() => {
    if (view === "directions") return [];
    const out: MapPin[] = [];
    const seen = new Set<string>();
    if (source && source.type !== "saved") {
      for (const p of results) {
        out.push({ id: p.key, lat: p.lat, lng: p.lng, label: p.name, kind: "result" });
        seen.add(p.key);
      }
    }
    for (const s of savedList) {
      const key = s.place_id || `saved:${s.id}`;
      if (seen.has(key)) continue;
      out.push({ id: key, lat: s.lat, lng: s.lng, label: s.name, kind: "saved" });
      seen.add(key);
    }
    if (selected && !seen.has(selected.key)) {
      out.push({ id: selected.key, lat: selected.lat, lng: selected.lng, label: selected.name, kind: "result" });
    }
    return out;
  }, [view, source, results, savedList, selected]);

  // Fit the map once per new result set.
  const fittedFor = useRef<string | null>(null);
  useEffect(() => {
    if (!source || resultsLoading || view !== "results") return;
    const sig = JSON.stringify(source) + results.length;
    if (fittedFor.current === sig || results.length === 0) return;
    fittedFor.current = sig;
    setFitKey((k) => k + 1);
    setMovedSinceSearch(false);
  }, [source, resultsLoading, results.length, view]);

  // ---- actions ----
  const flashToast = (msg: string) => {
    setToast(msg);
    window.setTimeout(() => setToast((t) => (t === msg ? null : t)), 4000);
  };

  const area = (): Area =>
    mapView ? { lat: mapView.lat, lng: mapView.lng, radius: mapView.radius } : { ...DEFAULT_CENTER, radius: 3000 };

  const runSearch = (q: string) => {
    setSource({ type: "search", q, ...area() });
    setSelected(null);
    setView("results");
    setSnap("half");
  };

  const runCategory = (type: string, label: string) => {
    if (source?.type === "nearby" && source.category === type && view === "results") {
      goHome();
      return;
    }
    setSource({ type: "nearby", category: type, label, ...area() });
    setSelected(null);
    setView("results");
    setSnap("half");
  };

  const showSaved = () => {
    if (source?.type === "saved" && view === "results") {
      goHome();
      return;
    }
    setSource({ type: "saved" });
    setSelected(null);
    setView("results");
    setSnap("half");
    fittedFor.current = null;
  };

  const searchThisArea = () => {
    if (!source || source.type === "saved") return;
    setSource({ ...source, ...area() });
    setMovedSinceSearch(false);
  };

  const selectPlace = (p: PlaceRef, opts: { pan?: boolean } = {}) => {
    setSelected(p);
    setView("details");
    setFocus({ lat: p.lat, lng: p.lng, zoom: opts.pan ? mapView?.zoom : undefined, key: Date.now() });
  };

  const goHome = () => {
    setView("home");
    setSource(null);
    setSelected(null);
    setRoute(null);
  };

  const goBack = () => {
    if (view === "directions") {
      setRoute(null);
      setView(selected ? "details" : source ? "results" : "home");
    } else if (view === "details") {
      setSelected(null);
      setView(source ? "results" : "home");
    } else {
      goHome();
    }
  };

  const locate = (then?: (pos: { lat: number; lng: number }) => void) => {
    if (!("geolocation" in navigator)) {
      flashToast("Location isn't available in this browser");
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const p = { lat: pos.coords.latitude, lng: pos.coords.longitude };
        setMe(p);
        setLocating(false);
        if (then) then(p);
        else setFocus({ ...p, zoom: 15, key: Date.now() });
      },
      () => {
        setLocating(false);
        flashToast("Couldn't get your location. Check the browser's location permission.");
      },
      { enableHighAccuracy: true, timeout: 10_000, maximumAge: 60_000 },
    );
  };

  useEffect(() => {
    let cancelled = false;
    navigator.permissions
      ?.query({ name: "geolocation" as PermissionName })
      .then((s) => {
        if (cancelled || s.state !== "granted") return;
        navigator.geolocation.getCurrentPosition((pos) => {
          if (cancelled) return;
          const p = { lat: pos.coords.latitude, lng: pos.coords.longitude };
          setMe(p);
          setFocus({ ...p, zoom: 14, key: Date.now() });
        });
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  const toggleSave = () => {
    if (!selected || createSaved.isPending || deleteSaved.isPending) return;
    const existing = savedMatch(selected);
    if (existing) {
      deleteSaved.mutate(existing.id, { onError: (e) => flashToast(errorMessage(e) ?? "Couldn't update saved places") });
      return;
    }
    createSaved.mutate(
      {
        name: selected.name,
        address: selected.address || undefined,
        lat: selected.lat,
        lng: selected.lng,
        place_id: selected.place_id || undefined,
      },
      { onError: (e) => flashToast(errorMessage(e) ?? "Couldn't update saved places") },
    );
  };

  // ---- routing ----
  const computeRoute = useComputeRoute();
  const routeToken = useRef(0);

  const plan = (next: { from: Endpoint | null; to: Endpoint | null; mode: TravelMode }) => {
    setFrom(next.from);
    setTo(next.to);
    setMode(next.mode);
    setRoute(null);
    setRouteError(null);
    const token = ++routeToken.current;
    if (!next.from || !next.to) return;
    computeRoute
      .mutateAsync({
        origin: { lat: next.from.lat, lng: next.from.lng },
        destination: { lat: next.to.lat, lng: next.to.lng },
        mode: next.mode,
        lang: navigator.language,
      })
      .then((r) => {
        if (token !== routeToken.current) return;
        setRoute(r);
        setFitKey((k) => k + 1);
      })
      .catch((e) => {
        if (token !== routeToken.current) return;
        setRouteError(isNoResult(e) ? "No route found for this travel mode." : (errorMessage(e) ?? "Couldn't get directions."));
      });
  };

  const openDirections = () => {
    if (!selected) return;
    setView("directions");
    setSnap("half");
    plan({
      from: me ? { label: "Your location", ...me, isMe: true } : null,
      to: { label: selected.name, lat: selected.lat, lng: selected.lng },
      mode,
    });
  };

  const mapRoute = useMemo(
    () =>
      view === "directions" && route && from && to
        ? { polyline: route.polyline, origin: { lat: from.lat, lng: from.lng }, destination: { lat: to.lat, lng: to.lng } }
        : null,
    [view, route, from, to],
  );

  // ---- layout ----
  const panelOpen = view !== "home";
  const sheetHeights = useMemo(
    () => ({ peek: 132, half: Math.round(shellHeight * 0.5), full: Math.max(shellHeight - 76, 240) }),
    [shellHeight],
  );
  const sheetPx = isDesktop || !panelOpen || !shellHeight ? 0 : sheetHeights[snap];

  const insets: MapInsets = useMemo(
    () => (isDesktop ? { left: PANEL_GAP + PANEL_WIDTH, bottom: 0 } : { left: 0, bottom: sheetPx }),
    [isDesktop, sheetPx],
  );

  const fabHidden = !isDesktop && panelOpen && snap === "full";
  useEffect(() => {
    const root = document.documentElement;
    root.style.setProperty("--fab-lift", `${fabHidden ? 0 : sheetPx}px`);
    if (fabHidden) root.dataset.fab = "hidden";
    else delete root.dataset.fab;
    return () => {
      root.style.removeProperty("--fab-lift");
      delete root.dataset.fab;
    };
  }, [sheetPx, fabHidden]);

  const onViewChange = useCallback(
    (v: MapView, byUser: boolean) => {
      setMapView(v);
      if (byUser) setMovedSinceSearch(true);
    },
    [],
  );

  const onPinClick = (pin: MapPin) => {
    const fromResults = results.find((r) => r.key === pin.id);
    const saved = savedList.find((s) => (s.place_id || `saved:${s.id}`) === pin.id);
    const place = fromResults ?? (saved ? fromSaved(saved) : selected?.key === pin.id ? selected : null);
    if (place) selectPlace(place, { pan: true });
  };

  useEffect(() => {
    if (!isDesktop) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape" || view === "home") return;
      const t = e.target as HTMLElement | null;
      if (t?.closest("[role=combobox],[role=listbox],[role=dialog]")) return;
      goBack();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const resultsTitle =
    source?.type === "search"
      ? `Results for “${source.q}”`
      : source?.type === "nearby"
        ? `${source.label} in this area`
        : "Saved places";
  const emptyText =
    source?.type === "saved"
      ? "Places you save will show up here."
      : "Nothing found here. Try zooming out or searching a nearby area.";

  const searchDisplay =
    view === "details" || view === "directions"
      ? (selected?.name ?? "")
      : source?.type === "search"
        ? source.q
        : "";

  const showSearchArea =
    movedSinceSearch && view === "results" && source !== null && source.type !== "saved" && !resultsLoading;

  const panelBody =
    view === "results" ? (
      <PlaceResults
        title={resultsTitle}
        places={results}
        loading={resultsLoading}
        error={resultsError}
        emptyText={emptyText}
        hoveredKey={hoveredKey}
        onHover={setHoveredKey}
        onSelect={(p) => selectPlace(p)}
      />
    ) : view === "details" && selected ? (
      <PlaceDetailsPanel
        key={selected.key}
        place={selected}
        isSaved={!!savedMatch(selected)}
        saving={createSaved.isPending || deleteSaved.isPending}
        onToggleSave={toggleSave}
        onDirections={openDirections}
      />
    ) : view === "directions" ? (
      <DirectionsPanel
        from={from}
        to={to}
        mode={mode}
        bias={bias}
        route={route}
        pending={computeRoute.isPending}
        error={routeError}
        locating={locating}
        onFrom={(e) => plan({ from: e, to, mode })}
        onTo={(e) => plan({ from, to: e, mode })}
        onSwap={() => plan({ from: to, to: from, mode })}
        onMode={(m) => plan({ from, to, mode: m })}
        onUseMyLocation={() => locate((p) => plan({ from: { label: "Your location", ...p, isMe: true }, to, mode }))}
      />
    ) : null;

  const chips = (
    <div
      role="toolbar"
      aria-label="Browse nearby"
      className="flex gap-2 overflow-x-auto no-scrollbar py-1 px-0.5 -mx-0.5"
    >
      <Chip active={source?.type === "saved" && view === "results"} onClick={showSaved} icon={<Bookmark className="size-4" />}>
        Saved
      </Chip>
      {CATEGORIES.map((c) => (
        <Chip
          key={c.type}
          active={source?.type === "nearby" && source.category === c.type && view !== "home"}
          onClick={() => runCategory(c.type, c.label)}
          icon={<c.icon className="size-4" />}
        >
          {c.label}
        </Chip>
      ))}
    </div>
  );

  const searchBox = (
    <PlaceSearchBox
      bias={bias}
      displayValue={searchDisplay}
      showBack={view !== "home"}
      onSubmit={runSearch}
      onPick={(p) => {
        setSource(null);
        selectPlace(fromSummary(p));
      }}
      onBack={goBack}
      onClear={goHome}
    />
  );

  return (
    <div ref={shellRef} className="absolute inset-0 md:inset-x-3 md:bottom-3 md:top-0 md:rounded-[28px] overflow-hidden bg-surface-sunken isolate">
      <PlacesMap
        pins={pins}
        selectedId={selected?.key}
        hoveredId={hoveredKey}
        route={mapRoute}
        me={me}
        initialCenter={DEFAULT_CENTER}
        focus={focus}
        fitKey={fitKey}
        insets={insets}
        locating={locating}
        onPinClick={onPinClick}
        onViewChange={onViewChange}
        onLocate={() => locate()}
      />

      {showSearchArea && (
        <button
          type="button"
          onClick={searchThisArea}
          className="absolute z-[600] left-1/2 -translate-x-1/2 inline-flex items-center gap-2 h-10 px-4 rounded-full bg-card shadow-soft-lg text-sm font-medium text-sky-ink hover:bg-sky-soft outline-none focus-visible:ring-2 focus-visible:ring-ring"
          style={{ top: isDesktop ? 80 : 128, marginLeft: isDesktop ? (PANEL_GAP + PANEL_WIDTH) / 2 : 0 }}
        >
          <RefreshCw className="size-4" /> Search this area
        </button>
      )}

      {toast && (
        <p
          role="status"
          className="absolute z-[700] left-1/2 -translate-x-1/2 bottom-6 max-w-[90%] rounded-full bg-foreground text-background text-sm px-4 py-2.5 shadow-soft-lg"
        >
          {toast}
        </p>
      )}

      {isDesktop ? (
        <>
          <div className="absolute z-[650] top-4 left-4 flex flex-col gap-3 max-h-[calc(100%-2rem)]" style={{ width: PANEL_WIDTH }}>
            {searchBox}
            {panelOpen && panelBody && (
              <div className="places-panel min-h-0 overflow-y-auto rounded-[28px] bg-card shadow-soft-lg p-4 chat-scroll">
                {panelBody}
              </div>
            )}
          </div>
          <div className="absolute z-[640] top-4 right-4" style={{ left: PANEL_WIDTH + 32 }}>
            <div className="flex justify-start">{chips}</div>
          </div>
        </>
      ) : (
        <>
          <div className="absolute z-[650] top-3 inset-x-3 flex flex-col gap-2">
            {searchBox}
            {chips}
          </div>
          {panelOpen && panelBody && (
            <BottomSheet heights={sheetHeights} snap={snap} onSnap={setSnap}>
              {panelBody}
            </BottomSheet>
          )}
        </>
      )}
    </div>
  );
}

function Chip({
  active,
  onClick,
  icon,
  children,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "shrink-0 inline-flex items-center gap-1.5 h-9 pl-3 pr-3.5 rounded-full text-sm shadow-soft transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
        active ? "bg-mint text-primary-foreground font-medium" : "bg-card text-foreground hover:bg-sky-soft",
      )}
    >
      <span className={active ? "text-primary-foreground" : "text-mint-ink"}>{icon}</span>
      {children}
    </button>
  );
}

function BottomSheet({
  heights,
  snap,
  onSnap,
  children,
}: {
  heights: Record<Snap, number>;
  snap: Snap;
  onSnap: (s: Snap) => void;
  children: React.ReactNode;
}) {
  const height = heights[snap];
  const maxHeight = heights.full;
  const [drag, setDrag] = useState<{ startY: number; startH: number; h: number } | null>(null);
  const shown = drag ? drag.h : height;

  const end = () => {
    if (!drag) return;
    const h = drag.h;
    setDrag(null);
    const nearest = (Object.entries(heights) as [Snap, number][]).reduce((a, b) =>
      Math.abs(b[1] - h) < Math.abs(a[1] - h) ? b : a,
    );
    onSnap(nearest[0]);
  };

  const cycle = () => onSnap(snap === "peek" ? "half" : snap === "half" ? "full" : "peek");

  return (
    <section
      aria-label="Place panel"
      className={cn(
        "absolute z-[650] inset-x-0 bottom-0 flex flex-col rounded-t-[28px] bg-card shadow-[0_-12px_40px_-12px_oklch(0.55_0.09_245/0.28)]",
        !drag && "transition-[height] duration-300 ease-[cubic-bezier(0.25,1,0.5,1)]",
      )}
      style={{ height: shown }}
    >
      <button
        type="button"
        aria-label={snap === "full" ? "Collapse panel" : "Expand panel"}
        onClick={cycle}
        onPointerDown={(e) => {
          (e.target as HTMLElement).setPointerCapture(e.pointerId);
          setDrag({ startY: e.clientY, startH: height, h: height });
        }}
        onPointerMove={(e) => {
          if (!drag) return;
          const h = Math.min(maxHeight, Math.max(96, drag.startH + (drag.startY - e.clientY)));
          setDrag({ ...drag, h });
        }}
        onPointerUp={(e) => {
          if (drag && Math.abs(drag.h - drag.startH) < 6) {
            setDrag(null);
            cycle();
            e.preventDefault();
            return;
          }
          end();
        }}
        onPointerCancel={end}
        className="shrink-0 grid place-items-center h-7 touch-none cursor-grab active:cursor-grabbing outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-t-[28px]"
      >
        <span className="w-10 h-1.5 rounded-full bg-input" />
      </button>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-6 overscroll-contain">{children}</div>
    </section>
  );
}

function isNoResult(e: unknown) {
  const status = (e as { status?: number } | null)?.status;
  return status === 404;
}

function useMediaQuery(query: string) {
  const [matches, setMatches] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia(query);
    const on = () => setMatches(mq.matches);
    on();
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [query]);
  return matches;
}

function useElementHeight(ref: React.RefObject<HTMLElement | null>) {
  const [h, setH] = useState(0);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) => setH(entry.contentRect.height));
    ro.observe(el);
    return () => ro.disconnect();
  }, [ref]);
  return h;
}
