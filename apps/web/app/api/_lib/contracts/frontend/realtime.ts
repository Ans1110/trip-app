import "server-only";
import { UpstreamRealtimeTicketResponse } from "../upstream";

export type RealtimeTicketView = {
  ticket: string;
  expires_in: number;
};

export const toRealtimeTicketView = (
  t: UpstreamRealtimeTicketResponse,
): RealtimeTicketView => ({
  ticket: t.ticket,
  expires_in: t.expires_in,
});
