import "server-only";
import {
  UpstreamJWKResponse,
  UpstreamSessionResponse,
  UpstreamTOTPSetupResponse,
  UpstreamUserResponse,
  UpstreamUserSession,
} from "./upstream";

export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  created_at: string;
};

export type SessionView = {
  user: User;
  expires_in: number;
  token_type?: string;
  requires_totp?: boolean;
};

export type TOTPSetupView = {
  secret: string;
  provisioning_url: string;
};

export type UserSessionView = {
  id: string;
  device_name: string;
  device_type: "web" | "ios" | "android" | "";
  ip_address: string;
  user_agent: string;
  last_active_at: string;
  expires_at: string;
  created_at: string;
};

export type JWKView = UpstreamJWKResponse;

export const toUser = (u: UpstreamUserResponse): User => ({
  id: u.id,
  email: u.email,
  name: u.name,
  avatar_url: u.avatar_url,
  created_at: u.created_at,
});

export const toSessionView = (s: UpstreamSessionResponse): SessionView => ({
  user: toUser(s.user),
  expires_in: s.expires_in,
  token_type: s.token_type,
  requires_totp: s.requires_totp,
});

export const toTOTPSetupView = (
  t: UpstreamTOTPSetupResponse,
): TOTPSetupView => ({
  secret: t.secret,
  provisioning_url: t.provisioning_url,
});

export const toUserSessionView = (s: UpstreamUserSession): UserSessionView => ({
  id: s.id,
  device_name: s.device_name,
  device_type: s.device_type,
  ip_address: s.ip_address,
  user_agent: s.user_agent,
  last_active_at: s.last_active_at,
  expires_at: s.expires_at,
  created_at: s.created_at,
});
