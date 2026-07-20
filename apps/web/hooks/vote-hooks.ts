"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { ApiError } from "@/lib/utils";
import {
  voteApi,
  type AddOptionInput,
  type CastVoteInput,
  type CreatePollInput,
  type Poll,
} from "@/lib/apis/vote-api";

export { errorMessage } from "@/hooks/trip-hooks";

export const voteKeys = {
  polls: (tripId: string) => ["vote", "polls", tripId] as const,
  poll: (pollId: string) => ["vote", "poll", pollId] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function usePolls(
  tripId: string | undefined,
  options?: QueryOpts<Poll[]>,
) {
  return useQuery<Poll[], ApiError>({
    queryKey: voteKeys.polls(tripId ?? ""),
    queryFn: ({ signal }) => voteApi.listPolls(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

export function usePoll(
  pollId: string | undefined,
  options?: QueryOpts<Poll>,
) {
  return useQuery<Poll, ApiError>({
    queryKey: voteKeys.poll(pollId ?? ""),
    queryFn: ({ signal }) => voteApi.getPoll(pollId!, signal),
    enabled: !!pollId,
    ...options,
  });
}

type CreatePollVars = { tripId: string; input: CreatePollInput };

export function useCreatePoll(options?: MutOpts<Poll, CreatePollVars>) {
  const qc = useQueryClient();
  return useMutation<Poll, ApiError, CreatePollVars>({
    mutationFn: ({ tripId, input }) => voteApi.createPoll(tripId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeletePollVars = { tripId: string; pollId: string };

export function useDeletePoll(options?: MutOpts<null, DeletePollVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeletePollVars>({
    mutationFn: ({ pollId }) => voteApi.deletePoll(pollId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      qc.removeQueries({ queryKey: voteKeys.poll(vars.pollId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type AddOptionVars = { tripId: string; pollId: string; input: AddOptionInput };

export function useAddOption(options?: MutOpts<Poll, AddOptionVars>) {
  const qc = useQueryClient();
  return useMutation<Poll, ApiError, AddOptionVars>({
    mutationFn: ({ pollId, input }) => voteApi.addOption(pollId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.setQueryData(voteKeys.poll(vars.pollId), data);
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteOptionVars = { tripId: string; pollId: string; optionId: string };

export function useDeleteOption(options?: MutOpts<Poll, DeleteOptionVars>) {
  const qc = useQueryClient();
  return useMutation<Poll, ApiError, DeleteOptionVars>({
    mutationFn: ({ optionId }) => voteApi.deleteOption(optionId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.setQueryData(voteKeys.poll(vars.pollId), data);
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type CastVoteVars = { tripId: string; pollId: string; input: CastVoteInput };

export function useCastVote(options?: MutOpts<Poll, CastVoteVars>) {
  const qc = useQueryClient();
  return useMutation<Poll, ApiError, CastVoteVars>({
    mutationFn: ({ pollId, input }) => voteApi.castVote(pollId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.setQueryData(voteKeys.poll(vars.pollId), data);
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type ClosePollVars = { tripId: string; pollId: string };

export function useClosePoll(options?: MutOpts<Poll, ClosePollVars>) {
  const qc = useQueryClient();
  return useMutation<Poll, ApiError, ClosePollVars>({
    mutationFn: ({ pollId }) => voteApi.closePoll(pollId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.setQueryData(voteKeys.poll(vars.pollId), data);
      qc.invalidateQueries({ queryKey: voteKeys.polls(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}
