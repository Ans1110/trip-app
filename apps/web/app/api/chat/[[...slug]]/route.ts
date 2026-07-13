import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { chatClient } from "../../_lib/clients/chat-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (req: NextRequest, auth: AuthArg) => Promise<ApiResult<unknown>>;

export async function GET(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const auth = buildAuth(req);
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const auth = buildAuth(req);
  const dispatch = matchPost(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const auth = buildAuth(req);
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /chat/rooms
  if (slug.length === 1 && slug[0] === "rooms") {
    return (_req, auth) => chatClient.listRooms(auth);
  }
  // GET /chat/rooms/{id}/messages
  if (slug.length === 3 && slug[0] === "rooms" && slug[2] === "messages") {
    return (req, auth) => {
      const query = parseListMessagesQuery(req);
      if (query instanceof Response) return responseToResult(query);
      return chatClient.listMessages(slug[1], query, auth);
    };
  }
  // GET /chat/rooms/{id}/receipts
  if (slug.length === 3 && slug[0] === "rooms" && slug[2] === "receipts") {
    return (_req, auth) => chatClient.listReadReceipts(slug[1], auth);
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /chat/rooms/dm
  if (slug.length === 2 && slug[0] === "rooms" && slug[1] === "dm") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return chatClient.ensureDM(body as never, auth);
    };
  }
  // POST /chat/trips/{tripId}/room
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "room") {
    return (_req, auth) => chatClient.ensureTripRoom(slug[1], auth);
  }
  // POST /chat/rooms/{id}/read
  if (slug.length === 3 && slug[0] === "rooms" && slug[2] === "read") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return chatClient.markRead(slug[1], body as never, auth);
    };
  }
  // POST /chat/rooms/{id}/ticket
  if (slug.length === 3 && slug[0] === "rooms" && slug[2] === "ticket") {
    return (_req, auth) => chatClient.issueRoomTicket(slug[1], auth);
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /chat/rooms/{id}
  if (slug.length === 2 && slug[0] === "rooms") {
    return (_req, auth) => chatClient.hideRoom(slug[1], auth);
  }
  return null;
};

const parseListMessagesQuery = (
  req: NextRequest,
): { before?: string; limit?: number } | NextResponse => {
  const sp = req.nextUrl.searchParams;
  const before = sp.get("before") ?? undefined;
  const limitRaw = sp.get("limit");
  let limit: number | undefined;
  if (limitRaw != null && limitRaw !== "") {
    const parsed = Number(limitRaw);
    if (!Number.isFinite(parsed) || parsed < 0) {
      return NextResponse.json(
        { code: 400, message: "invalid limit" },
        { status: 400 },
      );
    }
    limit = parsed;
  }
  return { before, limit };
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

const responseToResult = async (
  res: Response,
): Promise<ApiResult<unknown>> => {
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
      message: `unknown chat op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
