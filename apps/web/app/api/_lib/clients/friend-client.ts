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
  FriendBlockView,
  FriendRequestView,
  FriendView,
  toFriendBlockView,
  toFriendRequestView,
  toFriendView,
  toUserSummaryView,
  UserSummaryView,
} from "../contracts/frontend";
import {
  UpstreamBlockPayload,
  UpstreamBlockResponse,
  UpstreamFriendResponse,
  UpstreamRequestResponse,
  UpstreamSendFriendRequestPayload,
  UpstreamUserSummary,
} from "../contracts/upstream";

export type SendFriendRequestInput = UpstreamSendFriendRequestPayload;
export type BlockUserInput = UpstreamBlockPayload;

export type {
  FriendBlockView,
  FriendRequestView,
  FriendView,
  UserSummaryView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class FriendClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listFriends(auth: AuthCtx): Promise<ApiResult<FriendView[]>> {
    const res = await this.callProtected<UpstreamFriendResponse[]>(
      auth,
      (opts) => this.http.get<UpstreamFriendResponse[]>("/friends", opts),
    );
    return mapData(res, (rows) => rows.map(toFriendView));
  }

  async removeFriend(
    friendID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/friends/${encodeURIComponent(friendID)}`, opts),
    );
  }

  async searchUsers(
    query: string,
    auth: AuthCtx,
    limit?: number,
  ): Promise<ApiResult<UserSummaryView[]>> {
    const qs = new URLSearchParams({ q: query });
    if (limit && limit > 0) qs.set("limit", String(limit));
    const res = await this.callProtected<UpstreamUserSummary[]>(auth, (opts) =>
      this.http.get<UpstreamUserSummary[]>(`/friends/search?${qs}`, opts),
    );
    return mapData(res, (rows) => rows.map(toUserSummaryView));
  }

  async incomingRequests(
    auth: AuthCtx,
  ): Promise<ApiResult<FriendRequestView[]>> {
    const res = await this.callProtected<UpstreamRequestResponse[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamRequestResponse[]>("/friends/requests", opts),
    );
    return mapData(res, (rows) => rows.map(toFriendRequestView));
  }

  async outgoingRequests(
    auth: AuthCtx,
  ): Promise<ApiResult<FriendRequestView[]>> {
    const res = await this.callProtected<UpstreamRequestResponse[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamRequestResponse[]>(
          "/friends/requests/outgoing",
          opts,
        ),
    );
    return mapData(res, (rows) => rows.map(toFriendRequestView));
  }

  async sendRequest(
    input: SendFriendRequestInput,
    auth: AuthCtx,
  ): Promise<ApiResult<FriendRequestView>> {
    const res = await this.callProtected<UpstreamRequestResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamRequestResponse>(
          "/friends/requests",
          input,
          opts,
        ),
    );
    return mapData(res, toFriendRequestView);
  }

  async acceptRequest(
    requestID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/friends/requests/${encodeURIComponent(requestID)}/accept`,
        null,
        opts,
      ),
    );
  }

  async declineRequest(
    requestID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/friends/requests/${encodeURIComponent(requestID)}/decline`,
        null,
        opts,
      ),
    );
  }

  async cancelRequest(
    requestID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/friends/requests/${encodeURIComponent(requestID)}`,
        opts,
      ),
    );
  }

  async listBlocks(auth: AuthCtx): Promise<ApiResult<FriendBlockView[]>> {
    const res = await this.callProtected<UpstreamBlockResponse[]>(
      auth,
      (opts) => this.http.get<UpstreamBlockResponse[]>("/friends/blocks", opts),
    );
    return mapData(res, (rows) => rows.map(toFriendBlockView));
  }

  async blockUser(
    input: BlockUserInput,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>("/friends/blocks", input, opts),
    );
  }

  async unblockUser(targetID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/friends/blocks/${encodeURIComponent(targetID)}`,
        opts,
      ),
    );
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

export const friendClient = new FriendClient();
