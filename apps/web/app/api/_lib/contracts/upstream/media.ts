import "server-only";

export type UpstreamMediaPurpose =
  | "chat"
  | "avatar"
  | "cover"
  | "todo"
  | "album"
  | "album_thumb";

export type UpstreamInitUploadPayload = {
  purpose: UpstreamMediaPurpose;
  mime: string;
  bytes: number;
};

export type UpstreamInitUploadResponse = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type UpstreamCompleteUploadPayload = {
  upload_id: string;
};

export type UpstreamAssetResponse = {
  id: string;
  purpose: UpstreamMediaPurpose;
  mime: string;
  bytes: number;
  width?: number;
  height?: number;
  created_at: string;
};

export type UpstreamPresignedGetResponse = {
  url: string;
  expires_at: string;
};
