import "server-only";
import { createHash } from "node:crypto";

const required = (name: string, raw: string | undefined): string => {
  if (!raw) throw new Error(`${name} is not configured`);
  return raw;
};

const parseDurationMs = (raw: string | undefined, fallback: number): number => {
  if (!raw) return fallback;
  const match = /^(\d+)(ms|s|m)?$/.exec(raw.trim());
  if (!match) {
    const n = Number(raw);
    return Number.isFinite(n) && n > 0 ? n : fallback;
  }
  const n = Number(match[1]);
  const unit = match[2] ?? "ms";
  if (unit === "s") return n * 1000;
  if (unit === "m") return n * 60_000;
  return n;
};

const parseBool = (raw: string | undefined, fallback: boolean): boolean => {
  if (raw === undefined) return fallback;
  return raw.toLowerCase() === "true" || raw === "1";
};

export type CloudinarySignature = {
  cloud_name: string;
  api_key: string;
  timestamp: number;
  folder: string;
  signature: string;
  upload_url: string;
  timeout_ms: number;
};

export const buildUploadSignature = (): CloudinarySignature => {
  const cloudName = required("CLOUDINARY_NAME", process.env.CLOUDINARY_NAME);
  const apiKey = required("CLOUDINARY_API_KEY", process.env.CLOUDINARY_API_KEY);
  const apiSecret = required(
    "CLOUDINARY_API_SECRET",
    process.env.CLOUDINARY_API_SECRET,
  );
  const folder = required("CLOUDINARY_FOLDER", process.env.CLOUDINARY_FOLDER);
  const secure = parseBool(process.env.CLOUDINARY_SECURE, true);
  const timeoutMs = parseDurationMs(process.env.CLOUDINARY_TIMEOUT, 30_000);

  const timestamp = Math.floor(Date.now() / 1000);
  const toSign = `folder=${folder}&timestamp=${timestamp}`;
  const signature = createHash("sha1")
    .update(toSign + apiSecret)
    .digest("hex");

  const scheme = secure ? "https" : "http";
  return {
    cloud_name: cloudName,
    api_key: apiKey,
    timestamp,
    folder,
    signature,
    upload_url: `${scheme}://api.cloudinary.com/v1_1/${cloudName}/image/upload`,
    timeout_ms: timeoutMs,
  };
};
