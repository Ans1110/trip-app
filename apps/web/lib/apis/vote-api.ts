import { request } from "../utils";

export type PollType = "location" | "time" | "custom";
export type PollStatus = "open" | "closed";
export type PollResultVisibility = "always" | "after_vote" | "after_deadline";

export type PollOption = {
  id: string;
  text: string;
  meta?: Record<string, unknown>;
  added_by: string;
  sort_order: number;
  created_at: string;
  count: number;
  voters?: string[];
};

export type Poll = {
  id: string;
  trip_id: string;
  created_by: string;
  type: PollType;
  title: string;
  description?: string;
  is_anonymous: boolean;
  max_choices: number;
  allow_option_add: boolean;
  result_visibility: PollResultVisibility;
  deadline_at?: string;
  status: PollStatus;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  options: PollOption[];
  my_choices: string[];
  results_visible: boolean;
  total_voters: number;
};

export type CreatePollOptionInput = {
  text: string;
  meta?: Record<string, unknown>;
};

export type CreatePollInput = {
  type?: PollType;
  title: string;
  description?: string;
  is_anonymous?: boolean;
  max_choices?: number;
  allow_option_add?: boolean;
  result_visibility?: PollResultVisibility;
  deadline_at?: string;
  options?: CreatePollOptionInput[];
};

export type AddOptionInput = {
  text: string;
  meta?: Record<string, unknown>;
};

export type CastVoteInput = { option_ids: string[] };

export const voteApi = {
  listPolls: (tripId: string, signal?: AbortSignal) =>
    request<Poll[]>(`/api/vote/trips/${encodeURIComponent(tripId)}/polls`, {
      signal,
    }),
  createPoll: (tripId: string, input: CreatePollInput, signal?: AbortSignal) =>
    request<Poll>(`/api/vote/trips/${encodeURIComponent(tripId)}/polls`, {
      method: "POST",
      json: input,
      signal,
    }),
  getPoll: (pollId: string, signal?: AbortSignal) =>
    request<Poll>(`/api/vote/polls/${encodeURIComponent(pollId)}`, { signal }),
  deletePoll: (pollId: string, signal?: AbortSignal) =>
    request<null>(`/api/vote/polls/${encodeURIComponent(pollId)}`, {
      method: "DELETE",
      signal,
    }),
  addOption: (pollId: string, input: AddOptionInput, signal?: AbortSignal) =>
    request<Poll>(`/api/vote/polls/${encodeURIComponent(pollId)}/options`, {
      method: "POST",
      json: input,
      signal,
    }),
  deleteOption: (optionId: string, signal?: AbortSignal) =>
    request<Poll>(
      `/api/vote/polls/options/${encodeURIComponent(optionId)}`,
      { method: "DELETE", signal },
    ),
  castVote: (pollId: string, input: CastVoteInput, signal?: AbortSignal) =>
    request<Poll>(`/api/vote/polls/${encodeURIComponent(pollId)}/vote`, {
      method: "POST",
      json: input,
      signal,
    }),
  closePoll: (pollId: string, signal?: AbortSignal) =>
    request<Poll>(`/api/vote/polls/${encodeURIComponent(pollId)}/close`, {
      method: "POST",
      signal,
    }),
};
