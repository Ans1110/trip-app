import { request } from "../utils";

export type RealtimeTicket = {
  ticket: string;
  expires_in: number;
};

// realtimeApi only mints tickets.
export const realtimeApi = {
  issueTripTicket: (tripId: string, signal?: AbortSignal) =>
    request<RealtimeTicket>(
      `/api/realtime/trips/${encodeURIComponent(tripId)}/ticket`,
      { method: "POST", signal },
    ),
};
