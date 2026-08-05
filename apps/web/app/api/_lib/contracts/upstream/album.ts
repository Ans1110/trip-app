import "server-only";

export type UpstreamAlbumInitUploadItem = {
  mime: string;
  bytes: number;
};

export type UpstreamAlbumInitUploadPayload = {
  items: UpstreamAlbumInitUploadItem[];
};

export type UpstreamAlbumInitUploadSlot = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type UpstreamAlbumInitUploadResponse = {
  slots: UpstreamAlbumInitUploadSlot[];
};

export type UpstreamAlbumCompletePhotoPayload = {
  upload_id: string;
  caption?: string;
};

export type UpstreamAlbumUpdatePhotoPayload = {
  caption?: string;
};

export type UpstreamAlbumPhoto = {
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

export type UpstreamAlbumPhotoWithURLs = UpstreamAlbumPhoto & {
  original_url: string;
  thumb_small_url?: string;
  thumb_medium_url?: string;
};

export type UpstreamAlbumShareToken = {
  id: string;
  trip_id: string;
  created_by: string;
  expires_at?: string;
  revoked_at?: string;
  revoked_by?: string;
  last_accessed_at?: string;
  created_at: string;
};

export type UpstreamAlbumCreateSharePayload = {
  expires_at?: string;
};

export type UpstreamAlbumCreateShareResponse = {
  token: string;
  share: UpstreamAlbumShareToken;
};

export type UpstreamAlbumDownloadOriginal = {
  url: string;
  expires_at: string;
};

export type UpstreamAlbumPublicResponse = {
  photos: UpstreamAlbumPhotoWithURLs[];
  share: UpstreamAlbumShareToken;
};
