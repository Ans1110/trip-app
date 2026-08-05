import "server-only";
import {
  UpstreamAlbumCreateShareResponse,
  UpstreamAlbumDownloadOriginal,
  UpstreamAlbumInitUploadResponse,
  UpstreamAlbumInitUploadSlot,
  UpstreamAlbumPhoto,
  UpstreamAlbumPhotoWithURLs,
  UpstreamAlbumPublicResponse,
  UpstreamAlbumShareToken,
} from "../upstream";

export type AlbumInitUploadSlotView = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AlbumInitUploadView = {
  slots: AlbumInitUploadSlotView[];
};

export type AlbumPhotoView = {
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
};

export type AlbumPhotoWithURLsView = AlbumPhotoView & {
  original_url: string;
  thumb_small_url?: string;
  thumb_medium_url?: string;
};

export type AlbumShareTokenView = {
  id: string;
  trip_id: string;
  created_by: string;
  expires_at?: string;
  revoked_at?: string;
  revoked_by?: string;
  last_accessed_at?: string;
  created_at: string;
};

// AlbumCreateShareView carries the plaintext token exactly once — callers must
// surface it to the user immediately, since only the SHA-256 hash is persisted.
export type AlbumCreateShareView = {
  token: string;
  share: AlbumShareTokenView;
};

export type AlbumDownloadOriginalView = {
  url: string;
  expires_at: string;
};

export type AlbumPublicView = {
  photos: AlbumPhotoWithURLsView[];
  share: AlbumShareTokenView;
};

export const toAlbumInitUploadSlotView = (
  s: UpstreamAlbumInitUploadSlot,
): AlbumInitUploadSlotView => ({
  upload_id: s.upload_id,
  upload_url: s.upload_url,
  method: s.method,
  headers: s.headers,
  expires_at: s.expires_at,
});

export const toAlbumInitUploadView = (
  r: UpstreamAlbumInitUploadResponse,
): AlbumInitUploadView => ({
  slots: (r.slots ?? []).map(toAlbumInitUploadSlotView),
});

export const toAlbumPhotoView = (p: UpstreamAlbumPhoto): AlbumPhotoView => ({
  id: p.id,
  trip_id: p.trip_id,
  media_id: p.media_id,
  thumb_small_id: p.thumb_small_id,
  thumb_medium_id: p.thumb_medium_id,
  added_by: p.added_by,
  taken_at: p.taken_at,
  latitude: p.latitude,
  longitude: p.longitude,
  caption: p.caption,
  created_at: p.created_at,
});

export const toAlbumPhotoWithURLsView = (
  p: UpstreamAlbumPhotoWithURLs,
): AlbumPhotoWithURLsView => ({
  ...toAlbumPhotoView(p),
  original_url: p.original_url,
  thumb_small_url: p.thumb_small_url,
  thumb_medium_url: p.thumb_medium_url,
});

export const toAlbumShareTokenView = (
  s: UpstreamAlbumShareToken,
): AlbumShareTokenView => ({
  id: s.id,
  trip_id: s.trip_id,
  created_by: s.created_by,
  expires_at: s.expires_at,
  revoked_at: s.revoked_at,
  revoked_by: s.revoked_by,
  last_accessed_at: s.last_accessed_at,
  created_at: s.created_at,
});

export const toAlbumCreateShareView = (
  r: UpstreamAlbumCreateShareResponse,
): AlbumCreateShareView => ({
  token: r.token,
  share: toAlbumShareTokenView(r.share),
});

export const toAlbumDownloadOriginalView = (
  r: UpstreamAlbumDownloadOriginal,
): AlbumDownloadOriginalView => ({
  url: r.url,
  expires_at: r.expires_at,
});

export const toAlbumPublicView = (
  r: UpstreamAlbumPublicResponse,
): AlbumPublicView => ({
  photos: (r.photos ?? []).map(toAlbumPhotoWithURLsView),
  share: toAlbumShareTokenView(r.share),
});
