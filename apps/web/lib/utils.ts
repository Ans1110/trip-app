import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export type Envelope<T> = { code: number; message: string; data: T | null };

export class ApiError extends Error {
  status: number;
  code?: number;
  constructor(message: string, status: number, code?: number) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return formatApiError(err);
  return "Can't reach the server. Check your connection and try again.";
}

function formatApiError(err: ApiError): string | null {
  const raw = err.message?.trim();
  const generic =
    !raw ||
    /^unauthorized\.?$/i.test(raw) ||
    /^request failed \(\d+\)$/i.test(raw);
  if (!generic) return polish(raw);
  if (err.status === 401) return "unauthorized";
  if (err.status === 403) return "You don't have permission to do that.";
  if (err.status === 404) return "Not found.";
  if (err.status === 409) return "That conflicts with an existing record.";
  if (err.status === 413) return "That's too large to upload.";
  if (err.status === 429)
    return "Too many requests — wait a moment and try again.";
  if (err.status >= 500)
    return "Something went wrong on our end. Please try again.";
  return `Request failed (${err.status}).`;
}

const ACRONYMS: Record<string, string> = {
  mfa: "MFA",
  totp: "TOTP",
  oauth: "OAuth",
  url: "URL",
  api: "API",
  id: "ID",
  sms: "SMS",
  qr: "QR",
};

function polish(msg: string): string {
  const spaced = msg.replace(/\b[a-z][a-z0-9]*(?:_[a-z0-9]+)+\b/g, (m) =>
    m.replace(/_/g, " "),
  );
  const acronymized = spaced.replace(
    /\b[a-z]+\b/g,
    (word) => ACRONYMS[word] ?? word,
  );
  const cap = acronymized.charAt(0).toUpperCase() + acronymized.slice(1);
  return /[.!?…]$/.test(cap) ? cap : `${cap}.`;
}

export async function request<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const { json, headers, ...rest } = init;
  const res = await fetch(path, {
    credentials: "include",
    ...rest,
    headers: {
      ...(json !== undefined ? { "content-type": "application/json" } : {}),
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  });

  const body = (await res.json().catch(() => null)) as Envelope<T> | null;
  if (!res.ok) {
    throw new ApiError(
      body?.message ?? `Request failed (${res.status})`,
      res.status,
      body?.code,
    );
  }
  return (body?.data as T) ?? (null as T);
}
