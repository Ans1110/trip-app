import { request } from "../utils";

export type AlbumInitUploadItem = {
  mime: string;
  bytes: number;
};

export type AlbumInitUploadPayload = {
  items: AlbumInitUploadItem[];
};

export type AlbumInitUploadSlot = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AlbumInitUploadResponse = {
  slots: AlbumInitUploadSlot[];
};

export type AlbumCompletePhotoInput = {
  upload_id: string;
  caption?: string;
};

export type AlbumPhoto = {
  id: string;
  trip_id: string;
  media_id: string;
  thumb_small_id?: string;
  thumb_medium_id?: string;
  added_by: string;
  taken_at: string;
  latitude?: string;
  longitude?: string;
  caption: string;
  created_at: string;
  original_url: string;
  thumb_small_url?: string;
  thumb_medium_url?: string;
};

const path = (parts: string[]) => `/api/album/${parts.join("/")}`;

export const albumApi = {
  listPhotos: (tripId: string, signal?: AbortSignal) =>
    request<AlbumPhoto[]>(
      path(["trips", encodeURIComponent(tripId), "photos"]),
      { signal },
    ),

  initUpload: (
    tripId: string,
    input: AlbumInitUploadPayload,
    signal?: AbortSignal,
  ) =>
    request<AlbumInitUploadResponse>(
      path(["trips", encodeURIComponent(tripId), "init-upload"]),
      { method: "POST", json: input, signal },
    ),

  completePhoto: (
    tripId: string,
    input: AlbumCompletePhotoInput,
    signal?: AbortSignal,
  ) =>
    request<AlbumPhoto>(
      path(["trips", encodeURIComponent(tripId), "photos"]),
      { method: "POST", json: input, signal },
    ),
};
