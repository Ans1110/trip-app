"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  ApiError,
  friendApi,
  type BlockUserInput,
  type Friend,
  type FriendBlock,
  type FriendRequest,
  type SendRequestInput,
  type UserSummary,
} from "@/lib/friend-api";

export const friendKeys = {
  list: ["friends", "list"] as const,
  search: (q: string, limit?: number) =>
    ["friends", "search", q, limit ?? null] as const,
  incoming: ["friends", "requests", "incoming"] as const,
  outgoing: ["friends", "requests", "outgoing"] as const,
  blocks: ["friends", "blocks"] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useFriends(options?: QueryOpts<Friend[]>) {
  return useQuery<Friend[], ApiError>({
    queryKey: friendKeys.list,
    queryFn: ({ signal }) => friendApi.list(signal),
    ...options,
  });
}

export function useIncomingRequests(options?: QueryOpts<FriendRequest[]>) {
  return useQuery<FriendRequest[], ApiError>({
    queryKey: friendKeys.incoming,
    queryFn: ({ signal }) => friendApi.incomingRequests(signal),
    ...options,
  });
}

export function useOutgoingRequests(options?: QueryOpts<FriendRequest[]>) {
  return useQuery<FriendRequest[], ApiError>({
    queryKey: friendKeys.outgoing,
    queryFn: ({ signal }) => friendApi.outgoingRequests(signal),
    ...options,
  });
}

export function useBlocks(options?: QueryOpts<FriendBlock[]>) {
  return useQuery<FriendBlock[], ApiError>({
    queryKey: friendKeys.blocks,
    queryFn: ({ signal }) => friendApi.listBlocks(signal),
    ...options,
  });
}

export function useSearchUsers(
  query: string,
  limit?: number,
  options?: QueryOpts<UserSummary[]>,
) {
  const trimmed = query.trim();
  return useQuery<UserSummary[], ApiError>({
    queryKey: friendKeys.search(trimmed, limit),
    queryFn: ({ signal }) => friendApi.search(trimmed, limit, signal),
    enabled: trimmed.length > 0,
    staleTime: 10_000,
    ...options,
  });
}

export function useSendRequest(
  options?: MutOpts<FriendRequest, SendRequestInput>,
) {
  const qc = useQueryClient();
  return useMutation<FriendRequest, ApiError, SendRequestInput>({
    mutationFn: (input) => friendApi.sendRequest(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.outgoing });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAcceptRequest(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => friendApi.acceptRequest(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.incoming });
      qc.invalidateQueries({ queryKey: friendKeys.list });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useDeclineRequest(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => friendApi.declineRequest(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.incoming });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useCancelRequest(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => friendApi.cancelRequest(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.outgoing });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useRemoveFriend(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => friendApi.remove(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.list });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useBlockUser(options?: MutOpts<null, BlockUserInput>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, BlockUserInput>({
    mutationFn: (input) => friendApi.blockUser(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.blocks });
      qc.invalidateQueries({ queryKey: friendKeys.list });
      qc.invalidateQueries({ queryKey: friendKeys.incoming });
      qc.invalidateQueries({ queryKey: friendKeys.outgoing });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUnblockUser(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => friendApi.unblockUser(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: friendKeys.blocks });
      return options?.onSuccess?.(...args);
    },
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
