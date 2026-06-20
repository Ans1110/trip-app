import "server-only";

export type UpstreamRegisterRequest = {
  email: string;
  password: string;
  name: string;
};

export type UpstreamLoginRequest = {
  email: string;
  password: string;
  totp_code?: string;
};

export type UpstreamGoogleOAuthRequest = { id_token: string };
export type UpstreamGithubOAuthRequest = { code: string };
export type UpstreamFacebookOAuthRequest = { access_token: string };

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

export type UpstreamVerifyTOTPRequest = { totp_code: string };

export type UpstreamUserResponse = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  created_at: string;
};

export type UpstreamSessionResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type?: string;
  user: UpstreamUserResponse;
  requires_totp?: boolean;
  requires_verification?: boolean;
};

export type UpstreamTOTPSetupResponse = {
  secret: string;
  provisioning_url: string;
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

// ---- friend ----

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
