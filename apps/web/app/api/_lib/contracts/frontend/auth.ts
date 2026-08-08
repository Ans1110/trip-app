import "server-only";
import {
  UpstreamJWKResponse,
  UpstreamSessionResponse,
  UpstreamUserResponse,
  UpstreamUserSession,
} from "../upstream";

export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  mfa_enabled?: boolean;
  created_at: string;
};

export type SessionView = {
  user: User;
  expires_in: number;
  token_type?: string;
  requires_mfa?: boolean;
  requires_verification?: boolean;
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
  is_verified: u.is_verified,
  mfa_enabled: u.mfa_enabled,
  created_at: u.created_at,
});

export const toSessionView = (s: UpstreamSessionResponse): SessionView => ({
  user: toUser(s.user),
  expires_in: s.expires_in,
  token_type: s.token_type,
  requires_mfa: s.requires_mfa,
  requires_verification: s.requires_verification,
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
