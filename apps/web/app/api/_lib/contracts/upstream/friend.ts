import "server-only";

export type UpstreamUserSummary = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamFriendResponse = {
  user: UpstreamUserSummary;
  created_at: string;
};

export type UpstreamRequestResponse = {
  id: string;
  status: string;
  message?: string;
  sender: UpstreamUserSummary;
  receiver: UpstreamUserSummary;
  created_at: string;
  updated_at: string;
};

export type UpstreamBlockResponse = {
  user: UpstreamUserSummary;
  created_at: string;
};

export type UpstreamSendFriendRequestPayload = {
  receiver_id: string;
  message?: string;
};

export type UpstreamBlockPayload = {
  target_id: string;
};
