"use client";

import {
  useInfiniteQuery,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { feedApi, type Feed, type FeedPageQuery } from "@/lib/apis/feed-api";
import { ApiError } from "@/lib/utils";

export const feedKeys = {
  all: ["feed"] as const,
  list: (cursor?: string) => [...feedKeys.all, "list", cursor ?? null] as const,
  infinite: (limit?: number) =>
    [...feedKeys.all, "infinite", limit ?? null] as const,
};

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useFeed(query?: FeedPageQuery, options?: QueryOpts<Feed>) {
  return useQuery<Feed, ApiError>({
    queryKey: feedKeys.list(query?.cursor),
    queryFn: ({ signal }) => feedApi.list(query, signal),
    ...options,
  });
}

export function useInfiniteFeed(limit?: number) {
  return useInfiniteQuery<Feed, ApiError>({
    queryKey: feedKeys.infinite(limit),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      feedApi.list({ cursor: pageParam as string | undefined, limit }, signal),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
