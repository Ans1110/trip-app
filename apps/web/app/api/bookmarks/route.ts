import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { postClient } from "../_lib/clients/post-client";
import type { BookmarksQueryInput } from "../_lib/clients/post-client";
import type { ApiResult } from "../_lib/http-client";
import { extractRequestContext } from "../_lib/request-context";

export async function GET(req: NextRequest) {
  const auth = {
    ctx: extractRequestContext(req),
    signal: req.signal,
  };
  return toResponse(await postClient.listBookmarks(auth, readPageQuery(req)));
}

const readPageQuery = (req: NextRequest): BookmarksQueryInput => {
  const url = new URL(req.url);
  const out: BookmarksQueryInput = {};
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limitParam = url.searchParams.get("limit");
  if (limitParam) {
    const n = Number(limitParam);
    if (Number.isFinite(n) && n > 0) out.limit = n;
  }
  return out;
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
