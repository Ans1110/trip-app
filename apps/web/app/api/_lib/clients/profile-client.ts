import "server-only";
import { circuitOpenError } from "../errors";
import {
  ApiErr,
  ApiResult,
  Envelope,
  httpClient,
  HttpClient,
  HttpOptions,
} from "../http-client";
import { RequestContext } from "../request-context";
import { ensureCsrfToken, readSessionTokens } from "../session-store";
import {
  ListProfileUsersView,
  ProfileRecommendationView,
  ProfileView,
  toListProfileUsersView,
  toProfileRecommendationView,
  toProfileView,
} from "../contracts/frontend";
import {
  UpstreamListProfileUsersResponse,
  UpstreamProfilePageQuery,
  UpstreamProfileRecommendationResponse,
  UpstreamProfileResponse,
  UpstreamUpdateProfilePayload,
} from "../contracts/upstream";

export type UpdateProfileInput = UpstreamUpdateProfilePayload;
export type ProfilePageQueryInput = UpstreamProfilePageQuery;
export type RecommendationsQueryInput = { limit?: number };

export type {
  ListProfileUsersView,
  ProfileRecommendationView,
  ProfileView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class ProfileClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async getMyProfile(auth: AuthCtx): Promise<ApiResult<ProfileView>> {
    const res = await this.callProtected<UpstreamProfileResponse>(
      auth,
      (opts) => this.http.get<UpstreamProfileResponse>("/profiles/me", opts),
    );
    return mapData(res, toProfileView);
  }

  async updateMyProfile(
    input: UpdateProfileInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ProfileView>> {
    const res = await this.callProtected<UpstreamProfileResponse>(
      auth,
      (opts) =>
        this.http.patch<UpstreamProfileResponse>("/profiles/me", input, opts),
    );
    return mapData(res, toProfileView);
  }

  async getProfileByUsername(
    username: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ProfileView>> {
    const res = await this.callProtected<UpstreamProfileResponse>(
      auth,
      (opts) =>
        this.http.get<UpstreamProfileResponse>(
          `/profiles/by-username/${encodeURIComponent(username)}`,
          opts,
        ),
    );
    return mapData(res, toProfileView);
  }

  async follow(userID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/profiles/${encodeURIComponent(userID)}/follow`,
        null,
        opts,
      ),
    );
  }

  async unfollow(userID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/profiles/${encodeURIComponent(userID)}/follow`,
        opts,
      ),
    );
  }

  async listFollowers(
    userID: string,
    auth: AuthCtx,
    query: ProfilePageQueryInput = {},
  ): Promise<ApiResult<ListProfileUsersView>> {
    const qs = buildPageQuery(query);
    const base = `/profiles/${encodeURIComponent(userID)}/followers`;
    const path = qs ? `${base}?${qs}` : base;
    const res = await this.callProtected<UpstreamListProfileUsersResponse>(
      auth,
      (opts) => this.http.get<UpstreamListProfileUsersResponse>(path, opts),
    );
    return mapData(res, toListProfileUsersView);
  }

  async listFollowing(
    userID: string,
    auth: AuthCtx,
    query: ProfilePageQueryInput = {},
  ): Promise<ApiResult<ListProfileUsersView>> {
    const qs = buildPageQuery(query);
    const base = `/profiles/${encodeURIComponent(userID)}/following`;
    const path = qs ? `${base}?${qs}` : base;
    const res = await this.callProtected<UpstreamListProfileUsersResponse>(
      auth,
      (opts) => this.http.get<UpstreamListProfileUsersResponse>(path, opts),
    );
    return mapData(res, toListProfileUsersView);
  }

  async recommendations(
    auth: AuthCtx,
    query: RecommendationsQueryInput = {},
  ): Promise<ApiResult<ProfileRecommendationView>> {
    const qs = new URLSearchParams();
    if (query.limit && query.limit > 0) qs.set("limit", String(query.limit));
    const suffix = qs.toString();
    const path = suffix
      ? `/profiles/recommendations?${suffix}`
      : "/profiles/recommendations";
    const res = await this.callProtected<UpstreamProfileRecommendationResponse>(
      auth,
      (opts) =>
        this.http.get<UpstreamProfileRecommendationResponse>(path, opts),
    );
    return mapData(res, toProfileRecommendationView);
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
}

const buildPageQuery = (q: ProfilePageQueryInput): string => {
  const qs = new URLSearchParams();
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  return qs.toString();
};

const mapData = <TIn, TOut>(
  result: ApiResult<TIn>,
  map: (v: TIn) => TOut,
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

export const profileClient = new ProfileClient();
