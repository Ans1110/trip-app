import "server-only";
import { RequestContext } from "../request-context";
import { circuitOpenError } from "../errors";
import {
  ApiErr,
  ApiResult,
  Envelope,
  httpClient,
  HttpClient,
  HttpOptions,
} from "../http-client";
import { User } from "./auth-client";
import { ensureCsrfToken, readSessionTokens } from "../session-store";
import { UpstreamUserResponse } from "../contracts/upstream";
import { toUser } from "../contracts/frontend";

export type UpdateProfileInput = {
  name?: string;
  avatar_url?: string;
};

type AuthCtx = {
  ctx: RequestContext;
  signal?: AbortSignal;
};

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

export class UserClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async me(auth: AuthCtx): Promise<ApiResult<User>> {
    return this.callProtected<UpstreamUserResponse, User>(
      auth,
      (opts) => this.http.patch<UpstreamUserResponse>("/user/me", null, opts),
      toUser,
    );
  }

  async updateProfile(
    input: UpdateProfileInput,
    auth: AuthCtx,
  ): Promise<ApiResult<User>> {
    return this.callProtected<UpstreamUserResponse, User>(
      auth,
      (opts) => this.http.patch<UpstreamUserResponse>("/user/me", input, opts),
      toUser,
    );
  }

  private async callProtected<TUpstream, TView>(
    auth: AuthCtx,
    invoke: (opts: HttpOptions) => Promise<ApiResult<TUpstream>>,
    map: (v: TUpstream) => TView,
  ): Promise<ApiResult<TView>> {
    const tokens = await readSessionTokens();
    if (!tokens.accessToken) {
      return unauthorizedError("missing access token") as ApiResult<TView>;
    }
    const csrf = tokens.csrfToken ?? (await ensureCsrfToken(auth.ctx)) ?? "";
    if (!csrf) return csrfUnavailableError() as ApiResult<TView>;
    const deps: ProtectedDeps = {
      accessToken: tokens.accessToken,
      csrfToken: csrf,
    };
    const res = await invoke({
      ctx: auth.ctx,
      signal: auth.signal,
      accessToken: deps.accessToken,
      csrfToken: deps.csrfToken,
    });
    if (!res.ok) return res;
    return {
      ...res,
      data: map(res.data),
      envelope: res.envelope as unknown as Envelope<TView> | undefined,
    };
  }
}

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

function csrfUnavailableError(): ApiErr {
  const base = circuitOpenError();
  const message = "csrf token unavailable";
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

export const userClient = new UserClient();
