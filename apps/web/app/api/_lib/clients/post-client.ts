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
  CommentView,
  ListCommentsView,
  ListPostsView,
  PostView,
  toCommentView,
  toListCommentsView,
  toListPostsView,
  toPostView,
} from "../contracts/frontend";
import {
  UpstreamCommentResponse,
  UpstreamCreateCommentPayload,
  UpstreamCreatePostPayload,
  UpstreamListCommentsResponse,
  UpstreamListPostsQuery,
  UpstreamListPostsResponse,
  UpstreamPostResponse,
  UpstreamUpdateCommentPayload,
  UpstreamUpdatePostPayload,
} from "../contracts/upstream";

export type CreatePostInput = UpstreamCreatePostPayload;
export type UpdatePostInput = UpstreamUpdatePostPayload;
export type CreateCommentInput = UpstreamCreateCommentPayload;
export type UpdateCommentInput = UpstreamUpdateCommentPayload;
export type ListPostsQueryInput = UpstreamListPostsQuery;
export type PostPageQueryInput = { cursor?: string; limit?: number };

export type {
  CommentView,
  ListCommentsView,
  ListPostsView,
  PostAuthorSummaryView,
  PostView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class PostClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listPosts(
    auth: AuthCtx,
    query: ListPostsQueryInput = {},
  ): Promise<ApiResult<ListPostsView>> {
    const qs = buildListPostsQuery(query);
    const path = qs ? `/posts?${qs}` : "/posts";
    const res = await this.callProtected<UpstreamListPostsResponse>(
      auth,
      (opts) => this.http.get<UpstreamListPostsResponse>(path, opts),
    );
    return mapData(res, toListPostsView);
  }

  async getPost(id: string, auth: AuthCtx): Promise<ApiResult<PostView>> {
    const res = await this.callProtected<UpstreamPostResponse>(auth, (opts) =>
      this.http.get<UpstreamPostResponse>(
        `/posts/${encodeURIComponent(id)}`,
        opts,
      ),
    );
    return mapData(res, toPostView);
  }

  async createPost(
    input: CreatePostInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PostView>> {
    const res = await this.callProtected<UpstreamPostResponse>(auth, (opts) =>
      this.http.post<UpstreamPostResponse>("/posts", input, opts),
    );
    return mapData(res, toPostView);
  }

  async updatePost(
    id: string,
    input: UpdatePostInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PostView>> {
    const res = await this.callProtected<UpstreamPostResponse>(auth, (opts) =>
      this.http.patch<UpstreamPostResponse>(
        `/posts/${encodeURIComponent(id)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toPostView);
  }

  async deletePost(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/posts/${encodeURIComponent(id)}`, opts),
    );
  }

  async likePost(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/posts/${encodeURIComponent(id)}/likes`,
        null,
        opts,
      ),
    );
  }

  async unlikePost(id: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/posts/${encodeURIComponent(id)}/likes`, opts),
    );
  }

  async listComments(
    postID: string,
    auth: AuthCtx,
    query: PostPageQueryInput = {},
  ): Promise<ApiResult<ListCommentsView>> {
    const qs = buildPageQuery(query);
    const base = `/posts/${encodeURIComponent(postID)}/comments`;
    const path = qs ? `${base}?${qs}` : base;
    const res = await this.callProtected<UpstreamListCommentsResponse>(
      auth,
      (opts) => this.http.get<UpstreamListCommentsResponse>(path, opts),
    );
    return mapData(res, toListCommentsView);
  }

  async createComment(
    postID: string,
    input: CreateCommentInput,
    auth: AuthCtx,
  ): Promise<ApiResult<CommentView>> {
    const res = await this.callProtected<UpstreamCommentResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamCommentResponse>(
          `/posts/${encodeURIComponent(postID)}/comments`,
          input,
          opts,
        ),
    );
    return mapData(res, toCommentView);
  }

  async updateComment(
    postID: string,
    commentID: string,
    input: UpdateCommentInput,
    auth: AuthCtx,
  ): Promise<ApiResult<CommentView>> {
    const res = await this.callProtected<UpstreamCommentResponse>(
      auth,
      (opts) =>
        this.http.patch<UpstreamCommentResponse>(
          `/posts/${encodeURIComponent(postID)}/comments/${encodeURIComponent(commentID)}`,
          input,
          opts,
        ),
    );
    return mapData(res, toCommentView);
  }

  async deleteComment(
    postID: string,
    commentID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/posts/${encodeURIComponent(postID)}/comments/${encodeURIComponent(commentID)}`,
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

const buildListPostsQuery = (q: ListPostsQueryInput): string => {
  const qs = new URLSearchParams();
  if (q.author_id) qs.set("author_id", q.author_id);
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  return qs.toString();
};

const buildPageQuery = (q: PostPageQueryInput): string => {
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

export const postClient = new PostClient();
