import "server-only";

export type UpstreamPollType = "location" | "time" | "custom";
export type UpstreamPollStatus = "open" | "closed";
export type UpstreamPollResultVisibility =
  | "always"
  | "after_vote"
  | "after_deadline";

export type UpstreamPollOption = {
  id: string;
  text: string;
  meta?: Record<string, unknown>;
  added_by: string;
  sort_order: number;
  created_at: string;
  count: number;
  voters?: string[];
};

export type UpstreamPoll = {
  id: string;
  trip_id: string;
  created_by: string;
  type: UpstreamPollType;
  title: string;
  description?: string;
  is_anonymous: boolean;
  max_choices: number;
  allow_option_add: boolean;
  result_visibility: UpstreamPollResultVisibility;
  deadline_at?: string;
  status: UpstreamPollStatus;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  options: UpstreamPollOption[];
  my_choices: string[];
  results_visible: boolean;
  total_voters: number;
};

export type UpstreamPollOptionInput = {
  text: string;
  meta?: Record<string, unknown>;
};

export type UpstreamCreatePollPayload = {
  type?: UpstreamPollType | "";
  title: string;
  description?: string;
  is_anonymous?: boolean;
  max_choices?: number;
  allow_option_add?: boolean;
  result_visibility?: UpstreamPollResultVisibility | "";
  deadline_at?: string;
  options?: UpstreamPollOptionInput[];
};

export type UpstreamAddPollOptionPayload = {
  text: string;
  meta?: Record<string, unknown>;
};

export type UpstreamCastVotePayload = {
  option_ids: string[];
};
