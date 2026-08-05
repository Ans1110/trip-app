import "server-only";
import {
  UpstreamAssetResponse,
  UpstreamInitUploadResponse,
  UpstreamMediaPurpose,
  UpstreamPresignedGetResponse,
} from "../upstream";

export type MediaPurposeView = UpstreamMediaPurpose;

export type InitUploadView = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AssetView = {
  id: string;
  purpose: MediaPurposeView;
  mime: string;
  bytes: number;
  width?: number;
  height?: number;
  created_at: string;
};

export type PresignedGetView = {
  url: string;
  expires_at: string;
};

export const toInitUploadView = (
  r: UpstreamInitUploadResponse,
): InitUploadView => ({
  upload_id: r.upload_id,
  upload_url: r.upload_url,
  method: r.method,
  headers: r.headers,
  expires_at: r.expires_at,
});

export const toAssetView = (a: UpstreamAssetResponse): AssetView => ({
  id: a.id,
  purpose: a.purpose,
  mime: a.mime,
  bytes: a.bytes,
  width: a.width,
  height: a.height,
  created_at: a.created_at,
});

export const toPresignedGetView = (
  r: UpstreamPresignedGetResponse,
): PresignedGetView => ({
  url: r.url,
  expires_at: r.expires_at,
});
