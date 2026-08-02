"use client";

import { useCallback, useReducer } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  albumApi,
  type AlbumCreateShareInput,
  type AlbumCreateShareResponse,
  type AlbumPhoto,
  type AlbumPublicResponse,
  type AlbumShareToken,
  type AlbumUpdatePhotoInput,
} from "@/lib/apis/album-api";
import { putBinaryXhr } from "@/lib/album/put-xhr";
import {
  runUploadBatch,
  UploadError,
  type BatchEvent,
  type BatchResult,
  type UploadDeps,
  type UploadPhase,
  type UploadTicket,
} from "@/lib/album/upload";
import { convertHeicToJpeg, isHeic } from "@/lib/heic-convert";
import { ApiError } from "@/lib/utils";

export { errorMessage } from "@/hooks/trip-hooks";

export const albumKeys = {
  all: ["album"] as const,
  photos: (tripId: string) => ["album", "photos", tripId] as const,
  shares: (tripId: string) => ["album", "shares", tripId] as const,
  public: (token: string) => ["album", "public", token] as const,
};

export function invalidateAlbumForTrip(qc: QueryClient, tripId: string): void {
  void qc.invalidateQueries({ queryKey: albumKeys.photos(tripId) });
}

export function invalidateAlbumSharesForTrip(
  qc: QueryClient,
  tripId: string,
): void {
  void qc.invalidateQueries({ queryKey: albumKeys.shares(tripId) });
}

type QueryOpts<T> = Omit<UseQueryOptions<T, ApiError>, "queryKey" | "queryFn">;
type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

export function useAlbumPhotos(
  tripId: string | undefined,
  options?: QueryOpts<AlbumPhoto[]>,
) {
  return useQuery<AlbumPhoto[], ApiError>({
    queryKey: albumKeys.photos(tripId ?? ""),
    queryFn: ({ signal }) => albumApi.listPhotos(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

type UpdatePhotoVars = {
  photoId: string;
  tripId: string;
  input: AlbumUpdatePhotoInput;
};

export function useUpdatePhotoCaption(
  options?: MutOpts<AlbumPhoto, UpdatePhotoVars>,
) {
  const qc = useQueryClient();
  return useMutation<AlbumPhoto, ApiError, UpdatePhotoVars>({
    mutationFn: ({ photoId, input }) => albumApi.updatePhoto(photoId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateAlbumForTrip(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeletePhotoVars = { photoId: string; tripId: string };

export function useDeletePhoto(options?: MutOpts<null, DeletePhotoVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeletePhotoVars>({
    mutationFn: ({ photoId }) => albumApi.deletePhoto(photoId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateAlbumForTrip(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

export function useAlbumShares(
  tripId: string | undefined,
  options?: QueryOpts<AlbumShareToken[]>,
) {
  return useQuery<AlbumShareToken[], ApiError>({
    queryKey: albumKeys.shares(tripId ?? ""),
    queryFn: ({ signal }) => albumApi.listShareTokens(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

type CreateShareVars = { tripId: string; input?: AlbumCreateShareInput };

export function useCreateAlbumShare(
  options?: MutOpts<AlbumCreateShareResponse, CreateShareVars>,
) {
  const qc = useQueryClient();
  return useMutation<AlbumCreateShareResponse, ApiError, CreateShareVars>({
    mutationFn: ({ tripId, input }) =>
      albumApi.createShareToken(tripId, input ?? null),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateAlbumSharesForTrip(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type RevokeShareVars = { tokenId: string; tripId: string };

export function useRevokeAlbumShare(options?: MutOpts<null, RevokeShareVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, RevokeShareVars>({
    mutationFn: ({ tokenId }) => albumApi.revokeShareToken(tokenId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateAlbumSharesForTrip(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

export type PublicAlbumErrorKind = "invalid" | "gone" | "error";

// Backend collapses revoked + expired to 400 with a shared message, so we
// don't try to split them at the FE.
export function classifyPublicAlbumError(err: ApiError): PublicAlbumErrorKind {
  if (err.status === 404) return "invalid";
  if (err.status === 400) return "gone";
  return "error";
}

export function useAlbumPublic(
  token: string | undefined,
  options?: QueryOpts<AlbumPublicResponse>,
) {
  return useQuery<AlbumPublicResponse, ApiError>({
    queryKey: albumKeys.public(token ?? ""),
    queryFn: ({ signal }) => albumApi.resolvePublicAlbum(token!, signal),
    enabled: !!token,
    retry: (failureCount, err) => {
      if (err.status === 404 || err.status === 400) return false;
      return failureCount < 1;
    },
    ...options,
  });
}

// ---- Upload state (per-ticket, in-memory only) ----

export type UploadItemState = {
  ticket: UploadTicket;
  phase: UploadPhase;
  loaded: number;
  total: number;
  error?: UploadError;
};

type UploadState = Record<string, UploadItemState>;

type UploadAction =
  | { kind: "seed"; tickets: UploadTicket[] }
  | { kind: "event"; event: BatchEvent }
  | { kind: "clear"; ticketId: string }
  | { kind: "clearAll" };

function uploadReducer(state: UploadState, action: UploadAction): UploadState {
  switch (action.kind) {
    case "seed": {
      const next = { ...state };
      for (const t of action.tickets) {
        next[t.id] = {
          ticket: t,
          phase: "queued",
          loaded: 0,
          total: t.originalFile.size,
        };
      }
      return next;
    }
    case "event": {
      const cur = state[action.event.ticketId];
      if (!cur) return state;
      const merged = mergeEvent(cur, action.event);
      return { ...state, [action.event.ticketId]: merged };
    }
    case "clear": {
      if (!state[action.ticketId]) return state;
      const next = { ...state };
      delete next[action.ticketId];
      return next;
    }
    case "clearAll":
      return {};
  }
}

function mergeEvent(cur: UploadItemState, e: BatchEvent): UploadItemState {
  switch (e.kind) {
    case "phase":
      return { ...cur, phase: e.phase };
    case "progress":
      return { ...cur, loaded: e.loaded, total: e.total };
    case "success":
      return { ...cur, phase: "done", loaded: cur.total };
    case "error":
      return { ...cur, phase: "error", error: e.error };
  }
}

export type UseAlbumUploadResult = {
  items: UploadItemState[];
  isRunning: boolean;
  upload: (files: File[]) => Promise<BatchResult[]>;
  retry: (ticketId: string) => Promise<BatchResult[] | null>;
  clear: (ticketId: string) => void;
  clearAll: () => void;
};

export function useAlbumUpload(tripId: string): UseAlbumUploadResult {
  const qc = useQueryClient();
  const [state, dispatch] = useReducer(uploadReducer, {} as UploadState);

  const deps: UploadDeps = {
    isHeic,
    convertHeicToJpeg,
    initUpload: albumApi.initUpload,
    completePhoto: albumApi.completePhoto,
    putBinary: putBinaryXhr,
  };

  const runTickets = useCallback(
    async (tickets: UploadTicket[]): Promise<BatchResult[]> => {
      dispatch({ kind: "seed", tickets });
      const results = await runUploadBatch(tickets, tripId, deps, (event) =>
        dispatch({ kind: "event", event }),
      );
      if (results.some((r) => r.outcome.status === "success")) {
        invalidateAlbumForTrip(qc, tripId);
      }
      return results;
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [tripId, qc],
  );

  const upload = useCallback(
    (files: File[]) => {
      const tickets: UploadTicket[] = files.map((f) => ({
        id: makeTicketId(),
        originalFile: f,
      }));
      return runTickets(tickets);
    },
    [runTickets],
  );

  const retry = useCallback(
    (ticketId: string) => {
      const item = state[ticketId];
      if (!item) return Promise.resolve(null);
      return runTickets([item.ticket]);
    },
    [state, runTickets],
  );

  const items = Object.values(state).sort((a, b) =>
    a.ticket.id < b.ticket.id ? -1 : 1,
  );
  const isRunning = items.some(
    (i) => i.phase !== "done" && i.phase !== "error",
  );

  return {
    items,
    isRunning,
    upload,
    retry,
    clear: (ticketId) => dispatch({ kind: "clear", ticketId }),
    clearAll: () => dispatch({ kind: "clearAll" }),
  };
}

function makeTicketId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `t-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

// Exported for tests — pure reducer + merge, no React or crypto.
export const __test = { uploadReducer, mergeEvent, makeTicketId };
