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
  PollView,
  toPollView,
} from "../contracts/frontend";
import {
  UpstreamAddPollOptionPayload,
  UpstreamCastVotePayload,
  UpstreamCreatePollPayload,
  UpstreamPoll,
} from "../contracts/upstream";

export type CreatePollInput = UpstreamCreatePollPayload;
export type AddPollOptionInput = UpstreamAddPollOptionPayload;
export type CastVoteInput = UpstreamCastVotePayload;

export type {
  PollOptionView,
  PollResultVisibilityView,
  PollStatusView,
  PollTypeView,
  PollView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

// VoteClient proxies poll/option/ballot operations for the Go vote service.
// Same session-forwarding posture as ChatClient — the browser never holds the
// upstream JWT.
export class VoteClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listPolls(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView[]>> {
    const res = await this.callProtected<UpstreamPoll[]>(auth, (opts) =>
      this.http.get<UpstreamPoll[]>(
        `/trips/${encodeURIComponent(tripID)}/polls`,
        opts,
      ),
    );
    return mapData(res, (polls) => (polls ?? []).map(toPollView));
  }

  async createPoll(
    tripID: string,
    input: CreatePollInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.post<UpstreamPoll>(
        `/trips/${encodeURIComponent(tripID)}/polls`,
        input,
        opts,
      ),
    );
    return mapData(res, toPollView);
  }

  async getPoll(
    pollID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.get<UpstreamPoll>(
        `/polls/${encodeURIComponent(pollID)}`,
        opts,
      ),
    );
    return mapData(res, toPollView);
  }

  async deletePoll(
    pollID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/polls/${encodeURIComponent(pollID)}`, opts),
    );
  }

  async addOption(
    pollID: string,
    input: AddPollOptionInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.post<UpstreamPoll>(
        `/polls/${encodeURIComponent(pollID)}/options`,
        input,
        opts,
      ),
    );
    return mapData(res, toPollView);
  }

  async deleteOption(
    optionID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.delete<UpstreamPoll>(
        `/polls/options/${encodeURIComponent(optionID)}`,
        opts,
      ),
    );
    return mapData(res, toPollView);
  }

  async castVote(
    pollID: string,
    input: CastVoteInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.post<UpstreamPoll>(
        `/polls/${encodeURIComponent(pollID)}/vote`,
        input,
        opts,
      ),
    );
    return mapData(res, toPollView);
  }

  async closePoll(
    pollID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PollView>> {
    const res = await this.callProtected<UpstreamPoll>(auth, (opts) =>
      this.http.post<UpstreamPoll>(
        `/polls/${encodeURIComponent(pollID)}/close`,
        null,
        opts,
      ),
    );
    return mapData(res, toPollView);
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

export const voteClient = new VoteClient();
