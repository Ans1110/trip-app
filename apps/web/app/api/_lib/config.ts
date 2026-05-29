import "server-only";

export const UPSTREAM_API_BASE = (
  process.env.API_BASE_URL ?? "http://localhost:8080/api/v1"
).replace(/\/+$/, "");

export const UPSTREAM_TIMEOUT_MS = Number(
  process.env.UPSTREAM_TIMEOUT_MS ?? 15_000,
);

export const COOKIE_DOMAIN = process.env.AUTH_COOKIE_DOMAIN || undefined;

export const COOKIE_SECURE =
  process.env.AUTH_COOKIE_SECURE === "true" ||
  process.env.NODE_ENV === "production";

export const ACCESS_COOKIE = "access_token";
export const REFRESH_COOKIE = "refresh_token";
export const CSRF_COOKIE = "csrf_token";

export const AUTH_COOKIE_PATH = "/api/auth";

export const ACCESS_TTL_SECONDS = 60 * 15;
export const REFRESH_TTL_SECONDS = 60 * 60 * 24 * 30;
export const CSRF_TTL_SECONDS = 60 * 60 * 24;
