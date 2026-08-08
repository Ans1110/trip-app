import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { adminClient } from "../_lib/clients/admin-client";
import type { CreateReportInput } from "../_lib/clients/admin-client";
import type { ApiResult } from "../_lib/http-client";
import { extractRequestContext } from "../_lib/request-context";

export async function POST(req: NextRequest) {
  const body = await readJson(req);
  if (body instanceof Response) return body;
  const result = await adminClient.submitReport(
    (body ?? {}) as CreateReportInput,
    { ctx: extractRequestContext(req), signal: req.signal },
  );
  return toResponse(result);
}

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
