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

export type AlbumUpdatePhotoInput = {
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

export type AlbumShareToken = {
  id: string;
  trip_id: string;
  created_by: string;
  expires_at?: string;
  revoked_at?: string;
  revoked_by?: string;
  last_accessed_at?: string;
  created_at: string;
};

export type AlbumCreateShareInput = {
  expires_at?: string;
};

export type AlbumCreateShareResponse = {
  token: string;
  share: AlbumShareToken;
};

export type AlbumPublicResponse = {
  photos: AlbumPhoto[];
  share: AlbumShareToken;
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
    request<AlbumPhoto>(path(["trips", encodeURIComponent(tripId), "photos"]), {
      method: "POST",
      json: input,
      signal,
    }),

  updatePhoto: (
    photoId: string,
    input: AlbumUpdatePhotoInput,
    signal?: AbortSignal,
  ) =>
    request<AlbumPhoto>(path(["photos", encodeURIComponent(photoId)]), {
      method: "PATCH",
      json: input,
      signal,
    }),

  deletePhoto: (photoId: string, signal?: AbortSignal) =>
    request<null>(path(["photos", encodeURIComponent(photoId)]), {
      method: "DELETE",
      signal,
    }),

  listShareTokens: (tripId: string, signal?: AbortSignal) =>
    request<AlbumShareToken[]>(
      path(["trips", encodeURIComponent(tripId), "shares"]),
      { signal },
    ),

  createShareToken: (
    tripId: string,
    input: AlbumCreateShareInput | null,
    signal?: AbortSignal,
  ) =>
    request<AlbumCreateShareResponse>(
      path(["trips", encodeURIComponent(tripId), "shares"]),
      { method: "POST", json: input ?? {}, signal },
    ),

  revokeShareToken: (tokenId: string, signal?: AbortSignal) =>
    request<null>(path(["shares", encodeURIComponent(tokenId)]), {
      method: "DELETE",
      signal,
    }),

  resolvePublicAlbum: (token: string, signal?: AbortSignal) =>
    request<AlbumPublicResponse>(path(["public", encodeURIComponent(token)]), {
      signal,
    }),
};
