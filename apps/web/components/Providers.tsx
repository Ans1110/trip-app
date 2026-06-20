"use client";

import { useState, type ReactNode } from "react";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
  type Query,
} from "@tanstack/react-query";

import { ApiError } from "@/lib/auth-api";
import {
  isAuthPathname,
  sessionExpiredStore,
} from "@/lib/session-expired-store";

import { SessionExpiredModal } from "./auth/session-expired-modal";
import { FriendsFab } from "./friends/friends-fab";
import { SeasonProvider } from "./season-provider";

const shouldTriggerExpired = (
  err: unknown,
  query?: Query<unknown, unknown, unknown>,
): boolean => {
  if (!(err instanceof ApiError) || err.status !== 401) return false;
  if (typeof window !== "undefined" && isAuthPathname(window.location.pathname))
    return false;
  if (query && Array.isArray(query.queryKey) && query.queryKey[0] === "auth")
    return false;
  return true;
};

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        queryCache: new QueryCache({
          onError: (err, query) => {
            if (shouldTriggerExpired(err, query)) sessionExpiredStore.trigger();
          },
        }),
        mutationCache: new MutationCache({
          onError: (err) => {
            if (shouldTriggerExpired(err)) sessionExpiredStore.trigger();
          },
        }),
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
            retry: 1,
          },
          mutations: { retry: 0 },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <SeasonProvider>
        {children}
        <FriendsFab />
        <SessionExpiredModal />
      </SeasonProvider>
    </QueryClientProvider>
  );
}
