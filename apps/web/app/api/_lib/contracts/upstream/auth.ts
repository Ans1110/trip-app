import "server-only";

export type UpstreamRegisterRequest = {
  email: string;
  password: string;
  name: string;
};

export type UpstreamLoginRequest = {
  email: string;
  password: string;
  mfa_code?: string;
};

export type UpstreamGoogleOAuthRequest = { id_token: string };
export type UpstreamGithubOAuthRequest = { code: string };

export type UpstreamRefreshTokenRequest = { refresh_token: string };
export type UpstreamLogoutRequest = { refresh_token: string };

export type UpstreamVerifyEmailRequest = { token: string };
export type UpstreamResendVerificationRequest = { email: string };
export type UpstreamForgotPasswordRequest = { email: string };
export type UpstreamResetPasswordRequest = { token: string; password: string };

export type UpstreamChangePasswordRequest = {
  current_password: string;
  new_password: string;
};

export type UpstreamVerifyMFARequest = { mfa_code: string };

export type UpstreamUserResponse = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  mfa_enabled?: boolean;
  created_at: string;
};

export type UpstreamSessionResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type?: string;
  user: UpstreamUserResponse;
  requires_mfa?: boolean;
  requires_verification?: boolean;
};

export type UpstreamUserSession = {
  id: string;
  device_name: string;
  device_type: "web" | "ios" | "android" | "";
  ip_address: string;
  user_agent: string;
  last_active_at: string;
  expires_at: string;
  created_at: string;
};

export type UpstreamJWK = {
  kty: string;
  use?: string;
  kid: string;
  alg?: string;
  n?: string;
  e?: string;
};

export type UpstreamJWKResponse = {
  keys: UpstreamJWK[];
};
