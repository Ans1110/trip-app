import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { realtimeClient } from "../../../../_lib/clients/realtime-client";
import type { ApiResult } from "../../../../_lib/http-client";
import { extractRequestContext } from "../../../../_lib/request-context";

type RouteParams = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: RouteParams) {
  const { id } = await params;
  if (!id) return badRequest("trip id required");

  const result = await realtimeClient.issueTripTicket(id, {
    ctx: extractRequestContext(req),
    signal: req.signal,
  });
  return toResponse(result);
}

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

const badRequest = (message: string) =>
  NextResponse.json({ code: 400, message }, { status: 400 });
