import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { friendClient } from "../../_lib/clients/friend-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (req: NextRequest, auth: AuthArg) => Promise<ApiResult<unknown>>;

export async function GET(req: NextRequest, { params }: RouteParams) {
  const { slug } = await params;
  const auth = buildAuth(req);
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const { slug } = await params;
  const auth = buildAuth(req);
  const dispatch = matchPost(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const { slug } = await params;
  const auth = buildAuth(req);
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, auth));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /friends
  if (slug.length === 0) {
    return (_req, auth) => friendClient.listFriends(auth);
  }
  // GET /friends/search?q=&limit=
  if (slug.length === 1 && slug[0] === "search") {
    return (req, auth) => {
      const url = new URL(req.url);
      const q = url.searchParams.get("q") ?? "";
      const limitParam = url.searchParams.get("limit");
      const limit = limitParam ? Number(limitParam) : undefined;
      return friendClient.searchUsers(q, auth, limit);
    };
  }
  // GET /friends/requests
  if (slug.length === 1 && slug[0] === "requests") {
    return (_req, auth) => friendClient.incomingRequests(auth);
  }
  // GET /friends/requests/outgoing
  if (slug.length === 2 && slug[0] === "requests" && slug[1] === "outgoing") {
    return (_req, auth) => friendClient.outgoingRequests(auth);
  }
  // GET /friends/blocks
  if (slug.length === 1 && slug[0] === "blocks") {
    return (_req, auth) => friendClient.listBlocks(auth);
  }
  // GET /friends/invites
  if (slug.length === 1 && slug[0] === "invites") {
    return (_req, auth) => friendClient.listInvites(auth);
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /friends/requests
  if (slug.length === 1 && slug[0] === "requests") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return friendClient.sendRequest(body as never, auth);
    };
  }
  // POST /friends/requests/{id}/accept
  if (slug.length === 3 && slug[0] === "requests" && slug[2] === "accept") {
    return (_req, auth) => friendClient.acceptRequest(slug[1], auth);
  }
  // POST /friends/requests/{id}/decline
  if (slug.length === 3 && slug[0] === "requests" && slug[2] === "decline") {
    return (_req, auth) => friendClient.declineRequest(slug[1], auth);
  }
  // POST /friends/blocks
  if (slug.length === 1 && slug[0] === "blocks") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return friendClient.blockUser(body as never, auth);
    };
  }
  // POST /friends/invites
  if (slug.length === 1 && slug[0] === "invites") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return friendClient.createInvite(body as never, auth);
    };
  }
  // POST /friends/invites/accept
  if (slug.length === 2 && slug[0] === "invites" && slug[1] === "accept") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return friendClient.acceptInvite(body as never, auth);
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /friends/requests/{id}
  if (slug.length === 2 && slug[0] === "requests") {
    return (_req, auth) => friendClient.cancelRequest(slug[1], auth);
  }
  // DELETE /friends/blocks/{user_id}
  if (slug.length === 2 && slug[0] === "blocks") {
    return (_req, auth) => friendClient.unblockUser(slug[1], auth);
  }
  // DELETE /friends/invites/{id}
  if (slug.length === 2 && slug[0] === "invites") {
    return (_req, auth) => friendClient.revokeInvite(slug[1], auth);
  }
  // DELETE /friends/{user_id}
  if (slug.length === 1) {
    return (_req, auth) => friendClient.removeFriend(slug[0], auth);
  }
  return null;
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

// readJson may return a NextResponse on parse failure; wrap it so the dispatcher
// always returns an ApiResult that toResponse can render.
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
      message: `unknown friends op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
