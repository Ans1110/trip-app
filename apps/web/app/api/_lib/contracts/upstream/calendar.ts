import "server-only";

export type UpstreamEventVisibility = "private" | "room" | "friends" | "public";
export type UpstreamEventSource = "user" | "trip";
export type UpstreamEventType =
  | "general"
  | "trip"
  | "flight"
  | "hotel"
  | "meeting";

export type UpstreamEventResponse = {
  id: string;
  created_by: string;
  source_type: UpstreamEventSource;
  source_id?: string;
  event_type: UpstreamEventType;
  visibility: UpstreamEventVisibility;
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  time_zone?: string;
  all_day: boolean;
  color?: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type UpstreamCreateEventPayload = {
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: UpstreamEventType;
  visibility: "private" | "friends";
};

export type UpstreamUpdateEventPayload = {
  title?: string;
  description?: string;
  location?: string;
  start_at?: string;
  end_at?: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: UpstreamEventType;
  visibility?: "private" | "friends" | "public";
  version: number;
};

export type UpstreamListEventsQuery = {
  from?: string;
  to?: string;
};
