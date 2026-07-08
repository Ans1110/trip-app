import { request } from "../utils";

export type MediaPurpose = "chat" | "avatar" | "cover" | "todo";

export type InitUpload = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type Asset = {
  id: string;
  purpose: MediaPurpose;
  mime: string;
  bytes: number;
  width?: number;
  height?: number;
  created_at: string;
};

export type PresignedGet = {
  url: string;
  expires_at: string;
};

export type InitUploadInput = {
  purpose: MediaPurpose;
  mime: string;
  bytes: number;
};

export type CompleteUploadInput = {
  upload_id: string;
};

// putToPresignedUrl performs the browser-side PUT of the file body directly to
// object storage. It bypasses the BFF (no cookies, no /api/*) — the BFF only
// mints/verifies the handshake, it never streams bytes.
export async function putToPresignedUrl(
  init: InitUpload,
  body: Blob,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch(init.upload_url, {
    method: init.method,
    headers: init.headers,
    body,
    signal,
  });
  if (!res.ok) {
    throw new Error(`upload failed (${res.status})`);
  }
}

export const mediaApi = {
  initUpload: (input: InitUploadInput, signal?: AbortSignal) =>
    request<InitUpload>("/api/media/upload/init", {
      method: "POST",
      json: input,
      signal,
    }),

  completeUpload: (input: CompleteUploadInput, signal?: AbortSignal) =>
    request<Asset>("/api/media/upload/complete", {
      method: "POST",
      json: input,
      signal,
    }),

  getAsset: (id: string, signal?: AbortSignal) =>
    request<Asset>(`/api/media/${encodeURIComponent(id)}`, { signal }),

  presignRead: (id: string, signal?: AbortSignal) =>
    request<PresignedGet>(`/api/media/${encodeURIComponent(id)}/url`, {
      signal,
    }),

  softDelete: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/media/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  // uploadFile runs the full two-phase handshake: init → PUT → complete.
  // Returns the materialized Asset (server-observed size/mime/etag).
  uploadFile: async (
    purpose: MediaPurpose,
    file: File | Blob,
    signal?: AbortSignal,
  ): Promise<Asset> => {
    const mime =
      "type" in file && file.type ? file.type : "application/octet-stream";
    const init = await mediaApi.initUpload(
      { purpose, mime, bytes: file.size },
      signal,
    );
    await putToPresignedUrl(init, file, signal);
    return mediaApi.completeUpload({ upload_id: init.upload_id }, signal);
  },
};
