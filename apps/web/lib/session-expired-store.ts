"use client";

type Listener = () => void;

let expired = false;
const listeners = new Set<Listener>();

const emit = () => {
  for (const l of listeners) l();
};

export const sessionExpiredStore = {
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
  getSnapshot() {
    return expired;
  },
  getServerSnapshot() {
    return false;
  },
  trigger() {
    if (expired) return;
    expired = true;
    emit();
  },
  clear() {
    if (!expired) return;
    expired = false;
    emit();
  },
};

const AUTH_PATH_PREFIXES = [
  "/sign-in",
  "/sign-up",
  "/forgot-password",
  "/reset-password",
  "/verify-email",
];

export const isAuthPathname = (pathname: string): boolean =>
  AUTH_PATH_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
