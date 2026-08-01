import type {
  AlbumCompletePhotoInput,
  AlbumInitUploadPayload,
  AlbumInitUploadResponse,
  AlbumPhoto,
} from "../apis/album-api";
import {
  describeAlbumRejection,
  validateAlbumFile,
} from "../schemas/album-schemas";

export type UploadDeps = {
  isHeic(file: File): boolean;
  convertHeicToJpeg(file: File): Promise<File>;
  initUpload(
    tripId: string,
    input: AlbumInitUploadPayload,
    signal?: AbortSignal,
  ): Promise<AlbumInitUploadResponse>;
  putBinary(args: {
    url: string;
    method: string;
    headers?: Record<string, string>;
    body: Blob;
    signal?: AbortSignal;
    onProgress?: (loaded: number, total: number) => void;
  }): Promise<void>;
  completePhoto(
    tripId: string,
    input: AlbumCompletePhotoInput,
    signal?: AbortSignal,
  ): Promise<AlbumPhoto>;
};

export type UploadPhase =
  | "queued"
  | "converting"
  | "initiating"
  | "uploading"
  | "completing"
  | "done"
  | "error";

export type UploadTicket = {
  id: string; // stable id for React keys; assigned by caller
  originalFile: File;
};

export class UploadError extends Error {
  phase: UploadPhase;
  cause?: unknown;
  constructor(message: string, phase: UploadPhase, cause?: unknown) {
    super(message);
    this.phase = phase;
    this.cause = cause;
  }
}

export async function uploadOnePhoto(
  file: File,
  tripId: string,
  deps: UploadDeps,
  callbacks: {
    onPhase?: (phase: UploadPhase) => void;
    onProgress?: (loaded: number, total: number) => void;
    signal?: AbortSignal;
  } = {},
): Promise<AlbumPhoto> {
  const { onPhase, onProgress, signal } = callbacks;

  let effective = file;
  if (deps.isHeic(file)) {
    onPhase?.("converting");
    try {
      effective = await deps.convertHeicToJpeg(file);
    } catch (err) {
      throw new UploadError("HEIC conversion failed", "converting", err);
    }
  }

  const rejection = validateAlbumFile(effective);
  if (rejection) {
    throw new UploadError(describeAlbumRejection(rejection), "queued");
  }

  onPhase?.("initiating");
  let slotResp: AlbumInitUploadResponse;
  try {
    slotResp = await deps.initUpload(
      tripId,
      { items: [{ mime: effective.type, bytes: effective.size }] },
      signal,
    );
  } catch (err) {
    throw new UploadError(
      errorMessage(err, "Failed to reserve upload"),
      "initiating",
      err,
    );
  }
  const slot = slotResp.slots[0];
  if (!slot) {
    throw new UploadError("Upstream returned no upload slot", "initiating");
  }

  onPhase?.("uploading");
  try {
    await deps.putBinary({
      url: slot.upload_url,
      method: slot.method,
      headers: slot.headers,
      body: effective,
      signal,
      onProgress,
    });
  } catch (err) {
    throw new UploadError(errorMessage(err, "Upload failed"), "uploading", err);
  }

  onPhase?.("completing");
  try {
    const photo = await deps.completePhoto(
      tripId,
      { upload_id: slot.upload_id },
      signal,
    );
    onPhase?.("done");
    return photo;
  } catch (err) {
    throw new UploadError(
      errorMessage(err, "Failed to finalise photo"),
      "completing",
      err,
    );
  }
}

export type BatchEvent =
  | { kind: "phase"; ticketId: string; phase: UploadPhase }
  | { kind: "progress"; ticketId: string; loaded: number; total: number }
  | { kind: "success"; ticketId: string; photo: AlbumPhoto }
  | { kind: "error"; ticketId: string; error: UploadError };

export type BatchResult = {
  ticketId: string;
  outcome:
    | { status: "success"; photo: AlbumPhoto }
    | { status: "error"; error: UploadError };
};

export async function runUploadBatch(
  tickets: UploadTicket[],
  tripId: string,
  deps: UploadDeps,
  onEvent: (e: BatchEvent) => void,
  opts: { concurrency?: number; signal?: AbortSignal } = {},
): Promise<BatchResult[]> {
  const concurrency = Math.max(1, opts.concurrency ?? 3);
  const results: BatchResult[] = new Array(tickets.length);
  let cursor = 0;

  const runOne = async (idx: number) => {
    const ticket = tickets[idx];
    try {
      const photo = await uploadOnePhoto(ticket.originalFile, tripId, deps, {
        signal: opts.signal,
        onPhase: (phase) =>
          onEvent({ kind: "phase", ticketId: ticket.id, phase }),
        onProgress: (loaded, total) =>
          onEvent({ kind: "progress", ticketId: ticket.id, loaded, total }),
      });
      onEvent({ kind: "success", ticketId: ticket.id, photo });
      results[idx] = {
        ticketId: ticket.id,
        outcome: { status: "success", photo },
      };
    } catch (err) {
      const error =
        err instanceof UploadError
          ? err
          : new UploadError(errorMessage(err, "Upload failed"), "error", err);
      onEvent({ kind: "error", ticketId: ticket.id, error });
      onEvent({ kind: "phase", ticketId: ticket.id, phase: "error" });
      results[idx] = {
        ticketId: ticket.id,
        outcome: { status: "error", error },
      };
    }
  };

  const workers: Promise<void>[] = [];
  for (let i = 0; i < Math.min(concurrency, tickets.length); i++) {
    workers.push(
      (async () => {
        while (cursor < tickets.length) {
          const idx = cursor++;
          await runOne(idx);
        }
      })(),
    );
  }
  await Promise.all(workers);
  return results;
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}
