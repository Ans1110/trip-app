import "server-only";
import {
  ACCESS_COOKIE,
  ACCESS_TTL_SECONDS,
  AUTH_COOKIE_PATH,
  COOKIE_DOMAIN,
  COOKIE_SECURE,
  CSRF_COOKIE,
  REFRESH_COOKIE,
  REFRESH_TTL_SECONDS,
} from "./config";
import { NextResponse } from "next/server";

const options = (maxAge: number) => ({
  httpOnly: true,
  secure: COOKIE_SECURE,
  sameSite: "lax" as const,
  path: AUTH_COOKIE_PATH,
  domain: COOKIE_DOMAIN,
  maxAge,
});

export const setAccessCookie = (
  res: NextResponse,
  value: string,
  ttl?: number,
) => {
  res.cookies.set(
    ACCESS_COOKIE,
    value,
    options(ttl && ttl > 0 ? ttl : ACCESS_TTL_SECONDS),
  );
};

export const setRefreshCookie = (res: NextResponse, value: string) => {
  res.cookies.set(REFRESH_COOKIE, value, options(REFRESH_TTL_SECONDS));
};

export const setCsrfCookie = (res: NextResponse, value: string) => {
  res.cookies.set(CSRF_COOKIE, value, options(REFRESH_TTL_SECONDS));
};

export const clearSessionCookies = (res: NextResponse) => {
  res.cookies.delete({
    name: ACCESS_COOKIE,
    path: AUTH_COOKIE_PATH,
    domain: COOKIE_DOMAIN,
  });
  res.cookies.delete({
    name: REFRESH_COOKIE,
    path: AUTH_COOKIE_PATH,
    domain: COOKIE_DOMAIN,
  });
  res.cookies.delete({
    name: CSRF_COOKIE,
    path: AUTH_COOKIE_PATH,
    domain: COOKIE_DOMAIN,
  });
};
