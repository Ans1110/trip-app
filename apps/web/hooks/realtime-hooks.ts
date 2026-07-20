"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";

import { realtimeApi } from "@/lib/apis/realtime-api";
import { RealtimeClient, type RealtimeState } from "@/lib/realtime/client";
import {
  isServerMsg,
  type AckEvent,
  type ClientMsg,
  type CursorMoveEvent,
  type ErrorEvent,
  type ServerMsg,
  type WelcomeEvent,
} from "@/lib/realtime/protocol";

import { calendarKeys } from "./calendar-hooks";
import { tripKeys } from "./trip-hooks";
import { voteKeys } from "./vote-hooks";

const DEFAULT_WS_URL = "ws://localhost:8080";

export type UseTripRealtimeOptions = {
  enabled?: boolean;
  onWelcome?: (msg: WelcomeEvent) => void;
  onCursor?: (msg: CursorMoveEvent) => void;
  onAck?: (msg: AckEvent) => void;
  onServerError?: (msg: ErrorEvent) => void;
};

export type UseTripRealtimeResult = {
  state: RealtimeState;
  send: (msg: ClientMsg) => boolean;
};

export function useTripRealtime(
  tripId: string | null | undefined,
  options?: UseTripRealtimeOptions,
): UseTripRealtimeResult {
  const qc = useQueryClient();
  const [state, setState] = useState<RealtimeState>("idle");
  const clientRef = useRef<RealtimeClient<ClientMsg, ServerMsg> | null>(null);

  // Keep callbacks fresh without rebuilding the WS client on every render.
  const optsRef = useRef(options);
  useEffect(() => {
    optsRef.current = options;
  }, [options]);

  const enabled = (options?.enabled ?? true) && !!tripId;

  useEffect(() => {
    if (!enabled || !tripId) return;

    const client = new RealtimeClient<ClientMsg, ServerMsg>({
      wsUrl: buildTripWsUrl(tripId),
      fetchTicket: async (signal) => {
        const res = await realtimeApi.issueTripTicket(tripId, signal);
        return res.ticket;
      },
      parseMessage: (raw) => (isServerMsg(raw) ? raw : null),
      onMessage: (msg) => dispatchServerMsg(qc, tripId, msg, optsRef.current),
      onStateChange: setState,
      onError: (err) => {
        if (process.env.NODE_ENV !== "production") {
          // eslint-disable-next-line no-console
          console.warn("[realtime]", err);
        }
      },
    });

    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [enabled, tripId, qc]);

  const send = useCallback((msg: ClientMsg): boolean => {
    return clientRef.current?.send(msg) ?? false;
  }, []);

  return useMemo(() => ({ state, send }), [state, send]);
}

// dispatchServerMsg centralizes the "full payload -> setQueryData, notification
// -> invalidateQueries" policy for one place, so we can revisit strategy
// without hunting through call sites.
function dispatchServerMsg(
  qc: QueryClient,
  tripId: string,
  msg: ServerMsg,
  opts: UseTripRealtimeOptions | undefined,
): void {
  switch (msg.type) {
    case "WELCOME":
      opts?.onWelcome?.(msg);
      return;
    case "ACK":
      opts?.onAck?.(msg);
      return;
    case "ERROR":
      opts?.onServerError?.(msg);
      return;
    case "CURSOR_MOVE":
      // Never persisted; hand off to the caller so they can render presence.
      opts?.onCursor?.(msg);
      return;
    case "ITINERARY_UPDATE":
      // Payload is a full Itinerary but PascalCase (see protocol.ts) —
      // invalidate until server ships snake_case.
      void qc.invalidateQueries({ queryKey: tripKeys.itinerary(tripId) });
      return;
    case "TODO_UPDATE":
      void qc.invalidateQueries({ queryKey: tripKeys.todos(tripId) });
      return;
    case "CALENDAR_CREATED":
    case "CALENDAR_UPDATED":
    case "CALENDAR_DELETED":
      // The calendar event may belong to any list (mine/trip/friend), so
      // invalidate the whole calendar namespace and let the observers refetch.
      void qc.invalidateQueries({ queryKey: calendarKeys.all });
      return;
    case "VOTE_POLL_CREATED":
    case "VOTE_POLL_UPDATED":
    case "VOTE_POLL_CLOSED":
    case "VOTE_OPTION_ADDED":
    case "VOTE_CAST":
      void qc.invalidateQueries({ queryKey: voteKeys.polls(tripId) });
      void qc.invalidateQueries({ queryKey: voteKeys.poll(msg.poll.id) });
      return;
    case "VOTE_POLL_DELETED":
      void qc.invalidateQueries({ queryKey: voteKeys.polls(tripId) });
      qc.removeQueries({ queryKey: voteKeys.poll(msg.poll_id) });
      return;
  }
}

function buildTripWsUrl(tripId: string): string {
  const base = (process.env.NEXT_PUBLIC_WS_URL ?? DEFAULT_WS_URL).replace(
    /\/+$/,
    "",
  );
  return `${base}/ws/trips/${encodeURIComponent(tripId)}`;
}
