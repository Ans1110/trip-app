import { z } from "zod";

// Backend media/model.go PurposeAlbum: 50 MiB / {jpeg,png,webp,gif}. The FE
// gate matches — HEIC is out of scope for upload; heic-convert.ts converts to
// jpeg first, so by the time a File reaches this validator it must be a
// supported type.
export const ALBUM_MAX_BYTES = 50 * 1024 * 1024;

export const ALBUM_ACCEPTED_MIME = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
] as const;

export type AlbumAcceptedMime = (typeof ALBUM_ACCEPTED_MIME)[number];

export type AlbumFileRejection =
  | { kind: "mime"; got: string }
  | { kind: "size"; bytes: number; max: number }
  | { kind: "empty" };

export function validateAlbumFile(file: {
  type: string;
  size: number;
}): AlbumFileRejection | null {
  if (file.size === 0) return { kind: "empty" };
  if (file.size > ALBUM_MAX_BYTES) {
    return { kind: "size", bytes: file.size, max: ALBUM_MAX_BYTES };
  }
  if (!isAlbumAcceptedMime(file.type)) {
    return { kind: "mime", got: file.type };
  }
  return null;
}

export function isAlbumAcceptedMime(mime: string): mime is AlbumAcceptedMime {
  return (ALBUM_ACCEPTED_MIME as readonly string[]).includes(mime);
}

export function describeAlbumRejection(r: AlbumFileRejection): string {
  switch (r.kind) {
    case "empty":
      return "File is empty";
    case "size":
      return `File exceeds ${formatBytes(r.max)} limit`;
    case "mime":
      return `Unsupported type: ${r.got || "unknown"}`;
  }
}

export function formatBytes(n: number): string {
  const mb = n / (1024 * 1024);
  return `${mb % 1 === 0 ? mb : mb.toFixed(1)} MB`;
}

// Matches backend album.ErrCaptionTooLong.
export const ALBUM_CAPTION_MAX = 500;

export const albumCaptionSchema = z.object({
  caption: z
    .string()
    .trim()
    .max(
      ALBUM_CAPTION_MAX,
      `Caption must be ${ALBUM_CAPTION_MAX} characters or fewer`,
    ),
});

export type AlbumCaptionFormInput = z.infer<typeof albumCaptionSchema>;

export const albumShareCreateSchema = z
  .object({
    expires_at: z.string().trim().optional(),
  })
  .superRefine((data, ctx) => {
    if (!data.expires_at) return;
    const picked = parseExpiryEndOfDay(data.expires_at);
    if (!picked) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["expires_at"],
        message: "Invalid date",
      });
      return;
    }
    if (picked.getTime() <= Date.now()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["expires_at"],
        message: "Expiry must be in the future",
      });
    }
  });

export function parseExpiryEndOfDay(value: string): Date | null {
  const m = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const date = new Date(y, mo - 1, d, 23, 59, 59, 999);
  if (
    date.getFullYear() !== y ||
    date.getMonth() !== mo - 1 ||
    date.getDate() !== d
  ) {
    return null;
  }
  return date;
}

export type AlbumShareCreateFormInput = z.infer<typeof albumShareCreateSchema>;
