"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  profileApi,
  type Feed,
  type ListProfileUsers,
  type PageQuery,
  type Profile,
  type ProfileRecommendation,
  type UpdateProfileInput,
} from "@/lib/apis/profile-api";
import { ApiError } from "@/lib/utils";

export const profileKeys = {
  me: ["profile", "me"] as const,
  byUsername: (username: string) =>
    ["profile", "by-username", username] as const,
  followers: (userId: string, cursor?: string) =>
    ["profile", userId, "followers", cursor ?? null] as const,
  following: (userId: string, cursor?: string) =>
    ["profile", userId, "following", cursor ?? null] as const,
  recommendations: (limit?: number) =>
    ["profile", "recommendations", limit ?? null] as const,
  feed: (cursor?: string) => ["profile", "feed", cursor ?? null] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useMyProfile(options?: QueryOpts<Profile>) {
  return useQuery<Profile, ApiError>({
    queryKey: profileKeys.me,
    queryFn: ({ signal }) => profileApi.me(signal),
    ...options,
  });
}

export function useProfileByUsername(
  username: string,
  options?: QueryOpts<Profile>,
) {
  return useQuery<Profile, ApiError>({
    queryKey: profileKeys.byUsername(username),
    queryFn: ({ signal }) => profileApi.byUsername(username, signal),
    enabled: username.length > 0,
    ...options,
  });
}

export function useUpdateMyProfile(
  options?: MutOpts<Profile, UpdateProfileInput>,
) {
  const qc = useQueryClient();
  return useMutation<Profile, ApiError, UpdateProfileInput>({
    mutationFn: (input) => profileApi.updateMe(input),
    ...options,
    onSuccess: (...args) => {
      const [data] = args;
      qc.setQueryData(profileKeys.me, data);
      qc.invalidateQueries({ queryKey: profileKeys.byUsername(data.username) });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useFollow(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (userId) => profileApi.follow(userId),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["profile"] });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUnfollow(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (userId) => profileApi.unfollow(userId),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["profile"] });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useFollowers(
  userId: string,
  query?: PageQuery,
  options?: QueryOpts<ListProfileUsers>,
) {
  return useQuery<ListProfileUsers, ApiError>({
    queryKey: profileKeys.followers(userId, query?.cursor),
    queryFn: ({ signal }) => profileApi.followers(userId, query, signal),
    enabled: userId.length > 0,
    ...options,
  });
}

export function useFollowing(
  userId: string,
  query?: PageQuery,
  options?: QueryOpts<ListProfileUsers>,
) {
  return useQuery<ListProfileUsers, ApiError>({
    queryKey: profileKeys.following(userId, query?.cursor),
    queryFn: ({ signal }) => profileApi.following(userId, query, signal),
    enabled: userId.length > 0,
    ...options,
  });
}

export function useRecommendations(
  limit?: number,
  options?: QueryOpts<ProfileRecommendation>,
) {
  return useQuery<ProfileRecommendation, ApiError>({
    queryKey: profileKeys.recommendations(limit),
    queryFn: ({ signal }) => profileApi.recommendations(limit, signal),
    ...options,
  });
}

export function useFeed(query?: PageQuery, options?: QueryOpts<Feed>) {
  return useQuery<Feed, ApiError>({
    queryKey: profileKeys.feed(query?.cursor),
    queryFn: ({ signal }) => profileApi.feed(query, signal),
    ...options,
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
