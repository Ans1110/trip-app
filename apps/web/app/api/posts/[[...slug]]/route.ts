import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { postClient } from "../../_lib/clients/post-client";
import type {
  CreateCommentInput,
  CreatePostInput,
  ListPostsQueryInput,
  PostPageQueryInput,
  UpdateCommentInput,
  UpdatePostInput,
} from "../../_lib/clients/post-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (
  req: NextRequest,
  auth: AuthArg,
) => Promise<ApiResult<unknown>>;

export async function GET(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, buildAuth(req)));
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPost(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, buildAuth(req)));
}

export async function PATCH(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPatch(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, buildAuth(req)));
}

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, buildAuth(req)));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /posts
  if (slug.length === 0) {
    return (req, auth) => postClient.listPosts(auth, readListPostsQuery(req));
  }
  // GET /posts/:id
  if (slug.length === 1) {
    return (_req, auth) => postClient.getPost(slug[0], auth);
  }
  // GET /posts/:id/comments
  if (slug.length === 2 && slug[1] === "comments") {
    return (req, auth) =>
      postClient.listComments(slug[0], auth, readPageQuery(req));
  }
  // GET /posts/:id/comments/:comment_id/replies
  if (slug.length === 4 && slug[1] === "comments" && slug[3] === "replies") {
    return (req, auth) =>
      postClient.listReplies(slug[0], slug[2], auth, readPageQuery(req));
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /posts
  if (slug.length === 0) {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return postClient.createPost((body ?? {}) as CreatePostInput, auth);
    };
  }
  // POST /posts/:id/likes
  if (slug.length === 2 && slug[1] === "likes") {
    return (_req, auth) => postClient.likePost(slug[0], auth);
  }
  // POST /posts/:id/comments
  if (slug.length === 2 && slug[1] === "comments") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return postClient.createComment(
        slug[0],
        (body ?? {}) as CreateCommentInput,
        auth,
      );
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /posts/:id
  if (slug.length === 1) {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return postClient.updatePost(
        slug[0],
        (body ?? {}) as UpdatePostInput,
        auth,
      );
    };
  }
  // PATCH /posts/:id/comments/:comment_id
  if (slug.length === 3 && slug[1] === "comments") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return postClient.updateComment(
        slug[0],
        slug[2],
        (body ?? {}) as UpdateCommentInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /posts/:id
  if (slug.length === 1) {
    return (_req, auth) => postClient.deletePost(slug[0], auth);
  }
  // DELETE /posts/:id/likes
  if (slug.length === 2 && slug[1] === "likes") {
    return (_req, auth) => postClient.unlikePost(slug[0], auth);
  }
  // DELETE /posts/:id/comments/:comment_id
  if (slug.length === 3 && slug[1] === "comments") {
    return (_req, auth) => postClient.deleteComment(slug[0], slug[2], auth);
  }
  return null;
};

const readListPostsQuery = (req: NextRequest): ListPostsQueryInput => {
  const url = new URL(req.url);
  const out: ListPostsQueryInput = {};
  const author = url.searchParams.get("author_id");
  if (author) out.author_id = author;
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limitParam = url.searchParams.get("limit");
  if (limitParam) {
    const n = Number(limitParam);
    if (Number.isFinite(n) && n > 0) out.limit = n;
  }
  return out;
};

const readPageQuery = (req: NextRequest): PostPageQueryInput => {
  const url = new URL(req.url);
  const out: PostPageQueryInput = {};
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limitParam = url.searchParams.get("limit");
  if (limitParam) {
    const n = Number(limitParam);
    if (Number.isFinite(n) && n > 0) out.limit = n;
  }
  return out;
};

const buildAuth = (req: NextRequest): AuthArg => ({
  ctx: extractRequestContext(req),
  signal: req.signal,
});

const readJson = async (req: NextRequest): Promise<unknown> => {
  const len = req.headers.get("content-length");
  if (!len || len === "0") return null;
  try {
    return await req.json();
  } catch {
    return NextResponse.json(
      { code: 400, message: "invalid json body" },
      { status: 400 },
    );
  }
};

const toResponse = (result: ApiResult<unknown>): NextResponse => {
  if (result.ok) {
    return NextResponse.json(
      {
        code: result.envelope?.code ?? result.status,
        message: result.envelope?.message ?? "ok",
        data: result.data ?? null,
      },
      { status: result.status },
    );
  }
  return NextResponse.json(
    {
      code: result.envelope?.code ?? result.code ?? result.status,
      message: result.message,
      error: {
        category: result.error.category,
        retryable: result.error.retryable,
      },
    },
    { status: result.status >= 400 ? result.status : 500 },
  );
};

const responseToResult = async (res: Response): Promise<ApiResult<unknown>> => {
  const body = (await res.json().catch(() => ({}))) as {
    code?: number;
    message?: string;
  };
  return {
    ok: false,
    status: res.status,
    code: body.code ?? res.status,
    message: body.message ?? "bad request",
    error: {
      category: "validation",
      status: res.status,
      code: body.code ?? res.status,
      message: body.message ?? "bad request",
      retryable: false,
    },
    payload: body,
    raw: "",
  };
};

const notFound = (method: string, slug: string[]): NextResponse =>
  NextResponse.json(
    {
      code: 404,
      message: `unknown posts op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
