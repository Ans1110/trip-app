import "server-only";
import {
  UpstreamEventResponse,
  UpstreamEventSource,
  UpstreamEventType,
  UpstreamEventVisibility,
} from "../upstream";

export type EventVisibilityView = UpstreamEventVisibility;
export type EventSourceView = UpstreamEventSource;
export type EventTypeView = UpstreamEventType;

export type CalendarEventView = {
  id: string;
  created_by: string;
  source_type: EventSourceView;
  source_id?: string;
  event_type: EventTypeView;
  visibility: EventVisibilityView;
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

export const toCalendarEventView = (
  e: UpstreamEventResponse,
): CalendarEventView => ({
  id: e.id,
  created_by: e.created_by,
  source_type: e.source_type,
  source_id: e.source_id,
  event_type: e.event_type,
  visibility: e.visibility,
  title: e.title,
  description: e.description,
  location: e.location,
  start_at: e.start_at,
  end_at: e.end_at,
  time_zone: e.time_zone,
  all_day: e.all_day,
  color: e.color,
  version: e.version,
  created_at: e.created_at,
  updated_at: e.updated_at,
});
