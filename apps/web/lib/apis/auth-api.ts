import { request } from "../utils";

export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  is_admin?: boolean;
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

export const authApi = {
  signIn: (
    input: { email: string; password: string; mfa_code?: string },
    signal?: AbortSignal,
  ) =>
    request<SessionView>("/api/auth/login", {
      method: "POST",
      json: input,
      signal,
    }),
  requestMFAEnableCode: (signal?: AbortSignal) =>
    request<null>("/api/auth/mfa/enable/request", {
      method: "POST",
      signal,
    }),
  enableMFA: (input: { mfa_code: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/mfa/enable", {
      method: "POST",
      json: input,
      signal,
    }),
  requestMFADisableCode: (signal?: AbortSignal) =>
    request<null>("/api/auth/mfa/disable/request", {
      method: "POST",
      signal,
    }),
  disableMFA: (input: { mfa_code: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/mfa/disable", {
      method: "POST",
      json: input,
      signal,
    }),
  signUp: (
    input: { name: string; email: string; password: string },
    signal?: AbortSignal,
  ) =>
    request<SessionView>("/api/auth/register", {
      method: "POST",
      json: input,
      signal,
    }),
  forgotPassword: (input: { email: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/forgot-password", {
      method: "POST",
      json: input,
      signal,
    }),
  resetPassword: (
    input: { token: string; password: string },
    signal?: AbortSignal,
  ) =>
    request<null>("/api/auth/reset-password", {
      method: "POST",
      json: input,
      signal,
    }),
  verifyEmail: (input: { token: string }, signal?: AbortSignal) =>
    request<SessionView>("/api/auth/verify-email", {
      method: "POST",
      json: input,
      signal,
    }),
  resendVerification: (input: { email: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/resend-verification", {
      method: "POST",
      json: input,
      signal,
    }),
  oauthGoogle: (input: { id_token: string }, signal?: AbortSignal) =>
    request<SessionView>("/api/auth/oauth/google", {
      method: "POST",
      json: input,
      signal,
    }),
  oauthGithub: (input: { code: string }, signal?: AbortSignal) =>
    request<SessionView>("/api/auth/oauth/github", {
      method: "POST",
      json: input,
      signal,
    }),
  me: (signal?: AbortSignal) => request<User>("/api/auth/me", { signal }),
};
