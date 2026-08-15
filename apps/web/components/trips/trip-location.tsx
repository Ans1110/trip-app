"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { Loader2, LocateFixed, MapPin, Radio, RadioTower, X } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useRevokeTripPosition,
  useShareTripPosition,
  useTripMemberPositions,
} from "@/hooks/location-hooks";
import { useMyProfile } from "@/hooks/profile-hooks";
import { useRoom } from "@/hooks/trip-hooks";
import type { TripMemberPosition } from "@/lib/apis/location-api";
import type { RoomMember } from "@/lib/apis/trip-api";

import type { FocusTarget, LocationMapMe } from "./location-map";

const LocationMap = dynamic(() => import("./location-map"), { ssr: false });

const POLL_INTERVAL_MS = 30_000;

type ShareStatus =
  | { kind: "idle" }
  | { kind: "requesting" }
  | { kind: "denied"; message: string }
  | { kind: "error"; message: string };

type GeoState =
  | { kind: "unsupported" }
  | { kind: "pending" }
  | { kind: "ok"; lat: number; lng: number }
  | { kind: "denied" }
  | { kind: "error"; message: string };

export function TripLocation({ tripId }: { tripId: string }) {
  const meQuery = useMe();
  const profileQuery = useMyProfile({ enabled: !!meQuery.data });
  const roomQuery = useRoom(tripId);
  const positions = useTripMemberPositions(tripId, {
    refetchInterval: POLL_INTERVAL_MS,
    refetchOnWindowFocus: true,
    staleTime: POLL_INTERVAL_MS,
  });
  const share = useShareTripPosition();
  const revoke = useRevokeTripPosition();
  const [status, setStatus] = useState<ShareStatus>({ kind: "idle" });
  const [geo, setGeo] = useState<GeoState>({ kind: "pending" });
  const [focusTarget, setFocusTarget] = useState<FocusTarget | null>(null);
  const [endedNote, setEndedNote] = useState<string | null>(null);

  const focusOn = (lat: number, lng: number) =>
    setFocusTarget({ lat, lng, key: Date.now() });

  const myId = meQuery.data?.id;
  const memberIndex = useMemo(
    () => indexMembers(roomQuery.data?.members ?? []),
    [roomQuery.data?.members],
  );
  const positionList = positions.data ?? [];
  const iAmSharing = !!(myId && positionList.some((p) => p.user_id === myId));
  const iAmSharingRef = useRef(iAmSharing);
  iAmSharingRef.current = iAmSharing;

  const prevAuthedRef = useRef(false);
  useEffect(() => {
    const authed = !!meQuery.data;
    const wasAuthed = prevAuthedRef.current;
    prevAuthedRef.current = authed;
    if (wasAuthed && !authed) {
      if (iAmSharingRef.current) {
        revoke.mutate(tripId);
      }
      setEndedNote("You've been signed out. Sharing has stopped.");
    }
  }, [meQuery.data, revoke, tripId]);

  useEffect(() => {
    const handler = () => {
      if (!iAmSharingRef.current) return;
      // keepalive lets the DELETE complete even after the sign-out redirect
      // unmounts this component.
      fetch(`/api/location/trips/${encodeURIComponent(tripId)}/share`, {
        method: "DELETE",
        credentials: "include",
        keepalive: true,
      }).catch(() => {});
    };
    window.addEventListener("app:before-logout", handler);
    return () => window.removeEventListener("app:before-logout", handler);
  }, [tripId]);

  useEffect(() => {
    if (typeof navigator === "undefined" || !navigator.geolocation) {
      setGeo({ kind: "unsupported" });
      return;
    }
    const id = navigator.geolocation.watchPosition(
      (pos) =>
        setGeo({
          kind: "ok",
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
        }),
      (err) =>
        setGeo(
          err.code === err.PERMISSION_DENIED
            ? { kind: "denied" }
            : { kind: "error", message: err.message || "Location unavailable." },
        ),
      { enableHighAccuracy: true, maximumAge: 30_000, timeout: 15_000 },
    );
    return () => navigator.geolocation.clearWatch(id);
  }, []);

  const selfMember = myId ? memberIndex.get(myId) : undefined;
  const mapMe: LocationMapMe | null =
    geo.kind === "ok" && myId
      ? {
          id: myId,
          lat: geo.lat,
          lng: geo.lng,
          name:
            profileQuery.data?.name ??
            selfMember?.user.name ??
            meQuery.data?.name ??
            "You",
          avatarUrl:
            profileQuery.data?.avatar_url ||
            selfMember?.user.avatar_url ||
            meQuery.data?.avatar_url ||
            undefined,
        }
      : null;

  const shareCurrent = () => {
    if (geo.kind !== "ok") {
      setStatus({
        kind: geo.kind === "denied" ? "denied" : "error",
        message:
          geo.kind === "denied"
            ? "Location permission denied. Enable it in your browser to share."
            : geo.kind === "unsupported"
              ? "This browser doesn't support geolocation."
              : geo.kind === "error"
                ? geo.message
                : "Waiting for your location…",
      });
      return;
    }
    setStatus({ kind: "requesting" });
    setEndedNote(null);
    share.mutate(
      { tripId, input: { lat: geo.lat, lng: geo.lng } },
      {
        onSuccess: () => setStatus({ kind: "idle" }),
        onError: (err) =>
          setStatus({
            kind: "error",
            message: errorMessage(err) ?? "Failed to share location.",
          }),
      },
    );
  };

  const stopSharing = () => {
    setEndedNote(null);
    revoke.mutate(tripId, {
      onSuccess: () => setStatus({ kind: "idle" }),
      onError: (err) =>
        setStatus({
          kind: "error",
          message: errorMessage(err) ?? "Failed to stop sharing.",
        }),
    });
  };

  const busy = status.kind === "requesting" || share.isPending;

  return (
    <section className="flex flex-col gap-5">
      <header className="flex flex-col gap-2">
        <p
          className="text-[11px] font-medium tracking-[0.2em] uppercase"
          style={{ color: "var(--season-button)" }}
        >
          Location
        </p>
        <h2
          className="text-2xl tracking-tight text-[#ECEFEA]"
          style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
        >
          Where everyone is right now
        </h2>
        <p className="text-sm text-[#8B9A8E]">
          Your pin is only visible to you unless you tap Share. Sharing lasts 30
          minutes from each tap, then automatically stops.
        </p>
      </header>

      <div className="flex flex-wrap items-center gap-2">
        <span
          className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px]"
          style={{
            color: iAmSharing ? "var(--season-button)" : "#8B9A8E",
            backgroundColor: iAmSharing
              ? "color-mix(in srgb, var(--season-button) 15%, transparent)"
              : "transparent",
            border: `1px solid ${iAmSharing ? "transparent" : "#1F2A24"}`,
          }}
        >
          {iAmSharing ? (
            <RadioTower className="size-3.5" />
          ) : (
            <Radio className="size-3.5" />
          )}
          {iAmSharing ? "Sharing live" : "Not sharing"}
        </span>
        <span className="text-[11px] text-[#6B7A6F]">
          {positionList.length} member{positionList.length === 1 ? "" : "s"}{" "}
          on the map
        </span>
        <div className="ml-auto flex items-center gap-2">
          {iAmSharing && (
            <button
              type="button"
              onClick={stopSharing}
              disabled={revoke.isPending}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full border border-[#1F2A24] hover:bg-white/5 disabled:opacity-60 text-[#ECEFEA]"
            >
              {revoke.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <X className="size-3.5" />
              )}
              Stop sharing
            </button>
          )}
          <button
            type="button"
            onClick={shareCurrent}
            disabled={busy}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full text-[#0B100D] disabled:opacity-60"
            style={{ backgroundColor: "var(--season-button)" }}
          >
            {busy ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <MapPin className="size-3.5" />
            )}
            {iAmSharing ? "Update my location" : "Share my current location"}
          </button>
        </div>
      </div>

      {geo.kind === "pending" && (
        <p className="inline-flex items-center gap-1.5 text-xs text-[#8B9A8E]">
          <LocateFixed className="size-3.5 animate-pulse" />
          Locating you…
        </p>
      )}
      {geo.kind === "denied" && (
        <p className="text-xs text-[#FCA5A5]">
          Location permission denied. The map will show shared members only.
        </p>
      )}
      {geo.kind === "unsupported" && (
        <p className="text-xs text-[#FCA5A5]">
          This browser doesn&rsquo;t support geolocation.
        </p>
      )}
      {geo.kind === "error" && (
        <p className="text-xs text-[#FCA5A5]">{geo.message}</p>
      )}
      {status.kind === "denied" && (
        <p className="text-xs text-[#FCA5A5]">{status.message}</p>
      )}
      {status.kind === "error" && (
        <p className="text-xs text-[#FCA5A5]">{status.message}</p>
      )}
      {endedNote && (
        <p className="text-xs text-[#8B9A8E]">{endedNote}</p>
      )}

      <LocationMap
        me={mapMe}
        positions={positionList}
        membersById={memberIndex}
        meIsSharing={iAmSharing}
        focusTarget={focusTarget}
      />

      <SharingList
        positions={positionList}
        membersById={memberIndex}
        myId={myId}
        onFocus={focusOn}
      />
    </section>
  );
}

function SharingList({
  positions,
  membersById,
  myId,
  onFocus,
}: {
  positions: TripMemberPosition[];
  membersById: Map<string, RoomMember>;
  myId: string | undefined;
  onFocus: (lat: number, lng: number) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <h3
        className="text-[11px] font-medium tracking-[0.2em] uppercase"
        style={{ color: "#8B9A8E" }}
      >
        Sharing now · {positions.length}
      </h3>
      {positions.length === 0 ? (
        <p className="rounded-2xl border border-dashed border-[#1F2A24] p-6 text-sm text-[#8B9A8E] text-center">
          No one is sharing right now.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {positions.map((p) => {
            const member = membersById.get(p.user_id);
            const isSelf = p.user_id === myId;
            const name = member?.user.name ?? (isSelf ? "You" : "Member");
            const avatar = member?.user.avatar_url;
            const initial = (name || "?").slice(0, 1).toUpperCase();
            return (
              <li key={p.user_id}>
                <button
                  type="button"
                  onClick={() => onFocus(p.lat, p.lng)}
                  className="w-full grid grid-cols-[36px_1fr_auto] items-center gap-3 rounded-xl border border-[#1F2A24] bg-[#121814] px-4 py-3 text-left hover:bg-white/5 transition-colors"
                >
                  {avatar ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={avatar}
                      alt=""
                      width={36}
                      height={36}
                      loading="lazy"
                      className="rounded-full object-cover"
                      style={{ width: 36, height: 36 }}
                    />
                  ) : (
                    <span
                      className="flex items-center justify-center rounded-full text-xs text-[#ECEFEA]"
                      style={{ width: 36, height: 36, backgroundColor: "#1F2A24" }}
                    >
                      {initial}
                    </span>
                  )}
                  <div className="flex flex-col gap-0.5 min-w-0">
                    <span className="text-sm text-[#ECEFEA] truncate">
                      {name}
                      {isSelf && (
                        <span className="ml-2 text-[10px] uppercase tracking-widest text-[#8B9A8E]">
                          you
                        </span>
                      )}
                    </span>
                    <span className="text-xs text-[#8B9A8E] truncate">
                      {formatRelative(p.shared_at)}
                      {p.accuracy_m
                        ? ` · ±${Math.round(p.accuracy_m)} m`
                        : ""}
                    </span>
                  </div>
                  <MapPin className="size-4 text-[#8B9A8E]" />
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
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

function indexMembers(members: RoomMember[]): Map<string, RoomMember> {
  const m = new Map<string, RoomMember>();
  for (const it of members) m.set(it.user.id, it);
  return m;
}
