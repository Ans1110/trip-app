"use client";

import { useState, type ReactNode } from "react";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
  type Query,
} from "@tanstack/react-query";

import { authKeys } from "@/hooks/auth-hooks";
import type { User } from "@/lib/apis/auth-api";
import {
  isAuthPathname,
  sessionExpiredStore,
} from "@/store/session-expired-store";

import { SessionExpiredModal } from "./auth/session-expired-modal";
import { FriendsFab } from "./friends/friends-fab";
import { SeasonProvider } from "./season-provider";

const is401 = (err: unknown): boolean => {
  if (typeof err !== "object" || err === null) return false;
  const status = (err as { status?: unknown }).status;
  return status === 401;
};

const shouldTriggerExpired = (
  err: unknown,
  query?: Query<unknown, unknown, unknown>,
): boolean => {
  if (!is401(err)) return false;
  if (typeof window !== "undefined" && isAuthPathname(window.location.pathname))
    return false;
  if (query && Array.isArray(query.queryKey) && query.queryKey[0] === "auth")
    return false;
  return true;
};

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => {
    // eslint-disable-next-line prefer-const
    let client!: QueryClient;

    const handleAuthExpired = () => {
      const wasExpired = sessionExpiredStore.getSnapshot();
      sessionExpiredStore.trigger();
      if (wasExpired) return;
      void client.cancelQueries();
      client.setQueryData<User | null>(authKeys.me, null);
    };

    client = new QueryClient({
      queryCache: new QueryCache({
        onError: (err, query) => {
          if (shouldTriggerExpired(err, query)) handleAuthExpired();
        },
      }),
      mutationCache: new MutationCache({
        onError: (err) => {
          if (shouldTriggerExpired(err)) handleAuthExpired();
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
    });

    return client;
  });

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
