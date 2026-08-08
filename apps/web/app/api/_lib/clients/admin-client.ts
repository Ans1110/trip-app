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
  AdminListPostsView,
  AdminListReportsView,
  AdminListUsersView,
  AdminReportView,
  AdminUserView,
  ElevationView,
  toAdminListPostsView,
  toAdminListReportsView,
  toAdminListUsersView,
  toAdminReportView,
  toAdminUserView,
  toElevationView,
} from "../contracts/frontend";
import {
  UpstreamAdminListPostsQuery,
  UpstreamAdminListPostsResponse,
  UpstreamAdminListReportsQuery,
  UpstreamAdminListReportsResponse,
  UpstreamAdminListUsersQuery,
  UpstreamAdminListUsersResponse,
  UpstreamAdminReportResponse,
  UpstreamAdminUserResponse,
  UpstreamCreateReportPayload,
  UpstreamElevatePayload,
  UpstreamElevationResponse,
  UpstreamResolveReportPayload,
} from "../contracts/upstream";

export type ElevateInput = UpstreamElevatePayload;
export type CreateReportInput = UpstreamCreateReportPayload;
export type ResolveReportInput = UpstreamResolveReportPayload;
export type ListAdminUsersQuery = UpstreamAdminListUsersQuery;
export type ListAdminPostsQuery = UpstreamAdminListPostsQuery;
export type ListAdminReportsQuery = UpstreamAdminListReportsQuery;

export type {
  AdminListPostsView,
  AdminListReportsView,
  AdminListUsersView,
  AdminPostView,
  AdminReportView,
  AdminUserSummaryView,
  AdminUserView,
  ElevationView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class AdminClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async elevate(
    input: ElevateInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ElevationView>> {
    const res = await this.callProtected<UpstreamElevationResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamElevationResponse>(
          "/admin/auth/elevate",
          input,
          opts,
        ),
    );
    return mapData(res, toElevationView);
  }

  async adminLogout(auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/admin/auth/logout", null, opts),
    );
  }

  async submitReport(
    input: CreateReportInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AdminReportView>> {
    const res = await this.callProtected<UpstreamAdminReportResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamAdminReportResponse>("/reports", input, opts),
    );
    return mapData(res, toAdminReportView);
  }

  async listUsers(
    auth: AuthCtx,
    query: ListAdminUsersQuery = {},
  ): Promise<ApiResult<AdminListUsersView>> {
    const qs = buildListUsersQuery(query);
    const path = qs ? `/admin/users?${qs}` : "/admin/users";
    const res = await this.callProtected<UpstreamAdminListUsersResponse>(
      auth,
      (opts) => this.http.get<UpstreamAdminListUsersResponse>(path, opts),
    );
    return mapData(res, toAdminListUsersView);
  }

  async getUser(id: string, auth: AuthCtx): Promise<ApiResult<AdminUserView>> {
    const res = await this.callProtected<UpstreamAdminUserResponse>(
      auth,
      (opts) =>
        this.http.get<UpstreamAdminUserResponse>(
          `/admin/users/${encodeURIComponent(id)}`,
          opts,
        ),
    );
    return mapData(res, toAdminUserView);
  }

  async blockUser(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/admin/users/${encodeURIComponent(id)}/block`,
        null,
        opts,
      ),
    );
  }

  async unblockUser(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/admin/users/${encodeURIComponent(id)}/unblock`,
        null,
        opts,
      ),
    );
  }

  async deactivateUser(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/admin/users/${encodeURIComponent(id)}/deactivate`,
        null,
        opts,
      ),
    );
  }

  async listPosts(
    auth: AuthCtx,
    query: ListAdminPostsQuery = {},
  ): Promise<ApiResult<AdminListPostsView>> {
    const qs = buildListPostsQuery(query);
    const path = qs ? `/admin/posts?${qs}` : "/admin/posts";
    const res = await this.callProtected<UpstreamAdminListPostsResponse>(
      auth,
      (opts) => this.http.get<UpstreamAdminListPostsResponse>(path, opts),
    );
    return mapData(res, toAdminListPostsView);
  }

  async deletePost(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/admin/posts/${encodeURIComponent(id)}`, opts),
    );
  }

  async restorePost(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/admin/posts/${encodeURIComponent(id)}/restore`,
        null,
        opts,
      ),
    );
  }

  async deleteComment(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/admin/comments/${encodeURIComponent(id)}`,
        opts,
      ),
    );
  }

  async listReports(
    auth: AuthCtx,
    query: ListAdminReportsQuery = {},
  ): Promise<ApiResult<AdminListReportsView>> {
    const qs = buildListReportsQuery(query);
    const path = qs ? `/admin/reports?${qs}` : "/admin/reports";
    const res = await this.callProtected<UpstreamAdminListReportsResponse>(
      auth,
      (opts) => this.http.get<UpstreamAdminListReportsResponse>(path, opts),
    );
    return mapData(res, toAdminListReportsView);
  }

  async resolveReport(
    id: string,
    input: ResolveReportInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AdminReportView>> {
    const res = await this.callProtected<UpstreamAdminReportResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamAdminReportResponse>(
          `/admin/reports/${encodeURIComponent(id)}/resolve`,
          input,
          opts,
        ),
    );
    return mapData(res, toAdminReportView);
  }

  async dismissReport(
    id: string,
    input: ResolveReportInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AdminReportView>> {
    const res = await this.callProtected<UpstreamAdminReportResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamAdminReportResponse>(
          `/admin/reports/${encodeURIComponent(id)}/dismiss`,
          input,
          opts,
        ),
    );
    return mapData(res, toAdminReportView);
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

const buildListUsersQuery = (q: ListAdminUsersQuery): string => {
  const qs = new URLSearchParams();
  if (q.q) qs.set("q", q.q);
  if (q.status) qs.set("status", q.status);
  if (typeof q.is_blocked === "boolean") {
    qs.set("is_blocked", String(q.is_blocked));
  }
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  return qs.toString();
};

const buildListPostsQuery = (q: ListAdminPostsQuery): string => {
  const qs = new URLSearchParams();
  if (q.q) qs.set("q", q.q);
  if (q.status) qs.set("status", q.status);
  if (q.author_id) qs.set("author_id", q.author_id);
  if (typeof q.include_deleted === "boolean") {
    qs.set("include_deleted", String(q.include_deleted));
  }
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  return qs.toString();
};

const buildListReportsQuery = (q: ListAdminReportsQuery): string => {
  const qs = new URLSearchParams();
  if (q.status) qs.set("status", q.status);
  if (q.subject_type) qs.set("subject_type", q.subject_type);
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

export const adminClient = new AdminClient();
