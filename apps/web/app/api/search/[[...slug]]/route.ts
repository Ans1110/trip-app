import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { searchClient } from "../../_lib/clients/search-client";
import type { SearchPostsInput } from "../../_lib/clients/search-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

export async function GET(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  const auth = buildAuth(req);
  const built = dispatch(req, auth);
  if (built instanceof NextResponse) return built;
  return toResponse(await built);
}

const matchGet = (
  slug: string[],
):
  | ((req: NextRequest, auth: AuthArg) => Promise<ApiResult<unknown>> | NextResponse)
  | null => {
  // GET /search/posts?q=
  if (slug.length === 1 && slug[0] === "posts") {
    return (req, auth) => {
      const query = readSearchQuery(req);
      if (query instanceof NextResponse) return query;
      return searchClient.searchPosts(auth, query);
    };
  }
  return null;
};

const readSearchQuery = (req: NextRequest): SearchPostsInput | NextResponse => {
  const url = new URL(req.url);
  const q = url.searchParams.get("q");
  if (!q || !q.trim()) {
    return NextResponse.json(
      { code: 400, message: "query is required" },
      { status: 400 },
    );
  }
  const out: SearchPostsInput = { q: q.trim() };
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

const notFound = (method: string, slug: string[]): NextResponse =>
  NextResponse.json(
    {
      code: 404,
      message: `unknown search op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
