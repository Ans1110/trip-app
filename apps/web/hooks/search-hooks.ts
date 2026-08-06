"use client";

import {
  useInfiniteQuery,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  searchApi,
  type SearchPosts,
  type SearchPostsQuery,
} from "@/lib/apis/search-api";
import { ApiError } from "@/lib/utils";

export const searchKeys = {
  all: ["search"] as const,
  posts: (query: SearchPostsQuery) =>
    [
      ...searchKeys.all,
      "posts",
      query.q,
      query.cursor ?? null,
      query.limit ?? null,
    ] as const,
  infinitePosts: (q: string, limit?: number) =>
    [...searchKeys.all, "posts", "infinite", q, limit ?? null] as const,
};

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useSearchPosts(
  query: SearchPostsQuery,
  options?: QueryOpts<SearchPosts>,
) {
  const q = query.q.trim();
  return useQuery<SearchPosts, ApiError>({
    queryKey: searchKeys.posts({ ...query, q }),
    queryFn: ({ signal }) => searchApi.posts({ ...query, q }, signal),
    enabled: q.length > 0,
    ...options,
  });
}

export function useInfiniteSearchPosts(q: string, limit?: number) {
  const trimmed = q.trim();
  return useInfiniteQuery<SearchPosts, ApiError>({
    queryKey: searchKeys.infinitePosts(trimmed, limit),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      searchApi.posts(
        { q: trimmed, cursor: pageParam as string | undefined, limit },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    enabled: trimmed.length > 0,
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
