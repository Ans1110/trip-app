import "server-only";
import { rootLogger } from "./logger";
import {
  ACCESS_COOKIE,
  ACCESS_TTL_SECONDS,
  AUTH_COOKIE_PATH,
  COOKIE_DOMAIN,
  COOKIE_SECURE,
  CSRF_COOKIE,
  CSRF_TTL_SECONDS,
  REFRESH_COOKIE,
  REFRESH_TTL_SECONDS,
} from "./config";
import { cookies } from "next/headers";
import { RequestContext } from "./request-context";
import { httpClient } from "./http-client";
import { UpstreamSessionResponse } from "./contracts/upstream";

const log = rootLogger.child({ module: "session-store" });

export type SessionTokens = {
  accessToken: string | null;
  refreshToken: string | null;
  csrfToken: string | null;
};

const baseCookieOptions = (maxAge: number) => ({
  httpOnly: true,
  secure: COOKIE_SECURE,
  sameSite: "lax" as const,
  path: AUTH_COOKIE_PATH,
  domain: COOKIE_DOMAIN,
  maxAge,
});

export const readSessionTokens = async (): Promise<SessionTokens> => {
  const jar = await cookies();
  return {
    accessToken: jar.get(ACCESS_COOKIE)?.value ?? null,
    refreshToken: jar.get(REFRESH_COOKIE)?.value ?? null,
    csrfToken: jar.get(CSRF_COOKIE)?.value ?? null,
  };
};

export const ensureCsrfToken = async (
  ctx: RequestContext,
): Promise<string | null> => {
  const jar = await cookies();
  const existing = jar.get(CSRF_COOKIE)?.value;
  if (existing) return existing;
  const token = await httpClient.fetchCsrfToken(ctx);
  if (!token) return null;
  try {
    jar.set(CSRF_COOKIE, token, baseCookieOptions(CSRF_TTL_SECONDS));
  } catch (err) {
    log.debug("session.csrf_cookie_persist_skipped", {
      reason: err instanceof Error ? err.message : "unknown error",
    });
  }
  return token;
};

export const persistSession = async (
  session: UpstreamSessionResponse,
): Promise<void> => {
  const jar = await cookies();
  const ttl =
    session.expires_in && session.expires_in > 0
      ? session.expires_in
      : ACCESS_TTL_SECONDS;
  try {
    if (session.access_token) {
      jar.set(ACCESS_COOKIE, session.access_token, baseCookieOptions(ttl));
    }
    if (session.refresh_token) {
      jar.set(
        REFRESH_COOKIE,
        session.refresh_token,
        baseCookieOptions(REFRESH_TTL_SECONDS),
      );
    }
  } catch (err) {
    log.warn("session.persist_failed", {
      reason: err instanceof Error ? err.message : "unknown",
    });
  }
};

export const clearSession = async (): Promise<void> => {
  const jar = await cookies();
  const target = { path: AUTH_COOKIE_PATH, domain: COOKIE_DOMAIN };
  try {
    jar.delete({ name: ACCESS_COOKIE, ...target });
    jar.delete({ name: REFRESH_COOKIE, ...target });
    jar.delete({ name: CSRF_COOKIE, ...target });
  } catch (err) {
    log.warn("session.clear_failed", {
      reason: err instanceof Error ? err.message : "unknown",
    });
  }
};
