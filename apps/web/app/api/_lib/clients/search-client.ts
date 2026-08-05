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
import { SearchPostsView, toSearchPostsView } from "../contracts/frontend";
import {
  UpstreamSearchPostsQuery,
  UpstreamSearchPostsResponse,
} from "../contracts/upstream";

export type SearchPostsInput = UpstreamSearchPostsQuery;

export type { SearchPostsView } from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class SearchClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async searchPosts(
    auth: AuthCtx,
    query: SearchPostsInput,
  ): Promise<ApiResult<SearchPostsView>> {
    const qs = buildSearchQuery(query);
    const path = qs ? `/search/posts?${qs}` : "/search/posts";
    const res = await this.callProtected<UpstreamSearchPostsResponse>(
      auth,
      (opts) => this.http.get<UpstreamSearchPostsResponse>(path, opts),
    );
    return mapData(res, toSearchPostsView);
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

const buildSearchQuery = (q: SearchPostsInput): string => {
  const qs = new URLSearchParams();
  qs.set("q", q.q);
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

export const searchClient = new SearchClient();
