import "server-only";
import {
  UpstreamPoll,
  UpstreamPollOption,
  UpstreamPollResultVisibility,
  UpstreamPollStatus,
  UpstreamPollType,
} from "../upstream";

export type PollTypeView = UpstreamPollType;
export type PollStatusView = UpstreamPollStatus;
export type PollResultVisibilityView = UpstreamPollResultVisibility;

export type PollOptionView = {
  id: string;
  text: string;
  meta?: Record<string, unknown>;
  added_by: string;
  sort_order: number;
  created_at: string;
  count: number;
  voters?: string[];
};

export type PollView = {
  id: string;
  trip_id: string;
  created_by: string;
  type: PollTypeView;
  title: string;
  description?: string;
  is_anonymous: boolean;
  max_choices: number;
  allow_option_add: boolean;
  result_visibility: PollResultVisibilityView;
  deadline_at?: string;
  status: PollStatusView;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  options: PollOptionView[];
  my_choices: string[];
  results_visible: boolean;
  total_voters: number;
};

export const toPollOptionView = (o: UpstreamPollOption): PollOptionView => ({
  id: o.id,
  text: o.text,
  meta: o.meta,
  added_by: o.added_by,
  sort_order: o.sort_order,
  created_at: o.created_at,
  count: o.count ?? 0,
  voters: o.voters,
});

export const toPollView = (p: UpstreamPoll): PollView => ({
  id: p.id,
  trip_id: p.trip_id,
  created_by: p.created_by,
  type: p.type,
  title: p.title,
  description: p.description,
  is_anonymous: p.is_anonymous,
  max_choices: p.max_choices,
  allow_option_add: p.allow_option_add,
  result_visibility: p.result_visibility,
  deadline_at: p.deadline_at,
  status: p.status,
  closed_at: p.closed_at,
  created_at: p.created_at,
  updated_at: p.updated_at,
  options: (p.options ?? []).map(toPollOptionView),
  my_choices: p.my_choices ?? [],
  results_visible: p.results_visible,
  total_voters: p.total_voters ?? 0,
});
