import "server-only";
import {
  UpstreamChangePasswordRequest,
  UpstreamForgotPasswordRequest,
  UpstreamGithubOAuthRequest,
  UpstreamGoogleOAuthRequest,
  UpstreamJWKResponse,
  UpstreamLoginRequest,
  UpstreamRegisterRequest,
  UpstreamResendVerificationRequest,
  UpstreamResetPasswordRequest,
  UpstreamSessionResponse,
  UpstreamTOTPSetupResponse,
  UpstreamUserResponse,
  UpstreamUserSession,
  UpstreamVerifyEmailRequest,
  UpstreamVerifyTOTPRequest,
} from "../contracts/upstream";
import { RequestContext } from "../request-context";
import {
  ApiErr,
  ApiResult,
  Envelope,
  httpClient,
  HttpClient,
  HttpOptions,
} from "../http-client";
import { circuitOpenError } from "../errors";
import {
  SessionView,
  toSessionView,
  toTOTPSetupView,
  TOTPSetupView,
  toUser,
  toUserSessionView,
  User,
  UserSessionView,
} from "../contracts/frontend";
import {
  clearSession,
  ensureCsrfToken,
  persistSession,
  readSessionTokens,
} from "../session-store";

export type RegisterInput = UpstreamRegisterRequest;
export type LoginInput = UpstreamLoginRequest;
export type GoogleOAuthInput = UpstreamGoogleOAuthRequest;
export type GithubOAuthInput = UpstreamGithubOAuthRequest;
export type VerifyEmailInput = UpstreamVerifyEmailRequest;
export type ResendVerificationInput = UpstreamResendVerificationRequest;
export type ForgotPasswordInput = UpstreamForgotPasswordRequest;
export type ResetPasswordInput = UpstreamResetPasswordRequest;
export type ChangePasswordInput = UpstreamChangePasswordRequest;
export type VerifyTOTPInput = UpstreamVerifyTOTPRequest;

export type {
  JWKView,
  SessionView,
  TOTPSetupView,
  User,
  UserSessionView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class AuthClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async register(
    input: RegisterInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SessionView>> {
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/register",
      input,
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  async login(
    input: LoginInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SessionView>> {
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/login",
      input,
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  async oauthGoogle(
    input: GoogleOAuthInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SessionView>> {
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/oauth/google",
      input,
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  async oauthGithub(
    input: GithubOAuthInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SessionView>> {
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/oauth/github",
      input,
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  async refresh(auth: AuthCtx): Promise<ApiResult<SessionView>> {
    const tokens = await readSessionTokens();
    if (!tokens.refreshToken) return MISSING_SESSION as ApiResult<SessionView>;
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/refresh",
      { refresh_token: tokens.refreshToken },
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  async verifyEmail(
    input: VerifyEmailInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SessionView>> {
    const res = await this.http.post<UpstreamSessionResponse>(
      "/auth/verify-email",
      input,
      this.publicOpts(auth),
    );
    return this.captureSession(res);
  }

  resendVerification(input: ResendVerificationInput, auth: AuthCtx) {
    return this.http.post<null>(
      "/auth/resend-verification",
      input,
      this.publicOpts(auth),
    );
  }

  forgotPassword(input: ForgotPasswordInput, auth: AuthCtx) {
    return this.http.post<null>(
      "/auth/forgot-password",
      input,
      this.publicOpts(auth),
    );
  }

  resetPassword(input: ResetPasswordInput, auth: AuthCtx) {
    return this.http.post<null>(
      "/auth/reset-password",
      input,
      this.publicOpts(auth),
    );
  }

  jwk(auth: AuthCtx): Promise<ApiResult<UpstreamJWKResponse>> {
    return this.http.get<UpstreamJWKResponse>(
      "/auth/jwk",
      this.publicOpts(auth),
    );
  }

  async logout(auth: AuthCtx): Promise<ApiResult<null>> {
    const deps = await this.resolveProtected(auth);
    if ("error" in deps) return deps.error as ApiResult<null>;
    const tokens = await readSessionTokens();
    const res = await this.http.post<null>(
      "/auth/logout",
      { refresh_token: tokens.refreshToken ?? "" },
      this.protectedOpts(auth, deps.deps),
    );
    if (res.ok) await clearSession();
    return res;
  }

  async changePassword(
    input: ChangePasswordInput,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/auth/change-password", input, opts),
    );
  }

  async setupTOTP(auth: AuthCtx): Promise<ApiResult<TOTPSetupView>> {
    const res = await this.callProtected<UpstreamTOTPSetupResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamTOTPSetupResponse>(
          "/auth/totp/setup",
          null,
          opts,
        ),
    );
    return mapData(res, toTOTPSetupView);
  }

  async enableTOTP(
    input: VerifyTOTPInput,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/auth/totp/enable", input, opts),
    );
  }

  async disableTOTP(
    input: VerifyTOTPInput,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/auth/totp/disable", input, opts),
    );
  }

  async listSessions(auth: AuthCtx): Promise<ApiResult<UserSessionView[]>> {
    const res = await this.callProtected<UpstreamUserSession[]>(auth, (opts) =>
      this.http.get<UpstreamUserSession[]>("/auth/sessions", opts),
    );
    return mapData(res, (rows) => rows.map(toUserSessionView));
  }

  async deleteSession(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/auth/sessions/${id}`, opts),
    );
  }

  async me(auth: AuthCtx): Promise<ApiResult<User>> {
    const res = await this.callProtected<UpstreamUserResponse>(auth, (opts) =>
      this.http.get<UpstreamUserResponse>("/auth/me", opts),
    );
    return mapData(res, toUser);
  }

  async deactivate(auth: AuthCtx): Promise<ApiResult<null>> {
    const res = await this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/auth/deactivate", null, opts),
    );
    if (res.ok) await clearSession();
    return res;
  }

  async deleteAccount(auth: AuthCtx): Promise<ApiResult<null>> {
    const deps = await this.resolveProtected(auth);
    if ("error" in deps) return deps.error as ApiResult<null>;
    const res = await this.http.delete<null>(
      "/auth/delete",
      this.protectedOpts(auth, deps.deps),
    );
    if (res.ok) await clearSession();
    return res;
  }

  private publicOpts(auth: AuthCtx): HttpOptions {
    return {
      ctx: auth.ctx,
      signal: auth.signal,
    };
  }

  private protectedOpts(auth: AuthCtx, deps: ProtectedDeps): HttpOptions {
    return {
      ctx: auth.ctx,
      signal: auth.signal,
      accessToken: deps.accessToken,
      csrfToken: deps.csrfToken,
    };
  }

  private async resolveProtected(
    auth: AuthCtx,
  ): Promise<{ deps: ProtectedDeps } | { error: ApiErr }> {
    const tokens = await readSessionTokens();
    if (!tokens.accessToken) return { error: MISSING_SESSION };
    const csrf = tokens.csrfToken ?? (await ensureCsrfToken(auth.ctx)) ?? null;
    if (!csrf) return { error: MISSING_CSRF };
    return {
      deps: {
        accessToken: tokens.accessToken,
        csrfToken: csrf,
      },
    };
  }

  private async callProtected<T>(
    auth: AuthCtx,
    invoke: (opts: HttpOptions) => Promise<ApiResult<T>>,
  ): Promise<ApiResult<T>> {
    const resolved = await this.resolveProtected(auth);
    if ("error" in resolved) return resolved.error as ApiResult<T>;
    return invoke(this.protectedOpts(auth, resolved.deps));
  }

  private async captureSession(
    result: ApiResult<UpstreamSessionResponse>,
  ): Promise<ApiResult<SessionView>> {
    if (!result.ok) return result;
    const session = result.data;
    if (session && !session.requires_totp && !session.requires_verification) {
      await persistSession(session);
    }
    return {
      ...result,
      data: toSessionView(session),
      envelope: result.envelope as unknown as Envelope<SessionView> | undefined,
    };
  }
}

const mapData = <TIn, TOut>(
  result: ApiResult<TIn>,
  map: (V: TIn) => TOut,
): ApiResult<TOut> => {
  if (!result.ok) return result;
  return {
    ...result,
    data: map(result.data),
    envelope: result.envelope as unknown as Envelope<TOut> | undefined,
  };
};

function unauthorizedError(message: string): ApiErr {
  return {
    ok: false,
    status: 401,
    code: 401,
    message,
    error: {
      category: "auth",
      status: 401,
      code: 401,
      message,
      retryable: false,
    },
    payload: null,
    raw: "",
  };
}

function circuitGuardError(message: string): ApiErr {
  const base = circuitOpenError();
  return {
    ok: false,
    status: base.status,
    code: base.code,
    message,
    error: { ...base, message },
    payload: null,
    raw: "",
  };
}

export const authClient = new AuthClient();
