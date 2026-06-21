import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { tripClient } from "../../_lib/clients/trip-client";
import type {
  CreateItineraryInput,
  CreateTodoInput,
  CreateTripInput,
  ListTripsQueryInput,
  UpdateItineraryInput,
  UpdateTodoInput,
  UpdateTripInput,
} from "../../_lib/clients/trip-client";
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
  // GET /trips
  if (slug.length === 0) {
    return (req, auth) => {
      const q = readListTripsQuery(req);
      return tripClient.listTrips(auth, q);
    };
  }
  // GET /trips/:id
  if (slug.length === 1) {
    return (_req, auth) => tripClient.getTrip(slug[0], auth);
  }
  // GET /trips/:id/room
  if (slug.length === 2 && slug[1] === "room") {
    return (_req, auth) => tripClient.getRoom(slug[0], auth);
  }
  // GET /trips/:id/itinerary
  if (slug.length === 2 && slug[1] === "itinerary") {
    return (_req, auth) => tripClient.listItinerary(slug[0], auth);
  }
  // GET /trips/:id/todos
  if (slug.length === 2 && slug[1] === "todos") {
    return (_req, auth) => tripClient.listTodos(slug[0], auth);
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /trips
  if (slug.length === 0) {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return tripClient.createTrip(body as CreateTripInput, auth);
    };
  }
  // POST /trips/:id/room/leave
  if (slug.length === 3 && slug[1] === "room" && slug[2] === "leave") {
    return (_req, auth) => tripClient.leaveRoom(slug[0], auth);
  }
  // POST /trips/:id/room/code
  if (slug.length === 3 && slug[1] === "room" && slug[2] === "code") {
    return (_req, auth) => tripClient.regenerateCode(slug[0], auth);
  }
  // POST /trips/:id/itinerary
  if (slug.length === 2 && slug[1] === "itinerary") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return tripClient.createItinerary(
        slug[0],
        body as CreateItineraryInput,
        auth,
      );
    };
  }
  // POST /trips/:id/todos
  if (slug.length === 2 && slug[1] === "todos") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return tripClient.createTodo(slug[0], body as CreateTodoInput, auth);
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /trips/:id
  if (slug.length === 1) {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      if (isCompletedStatusInPayload(body)) {
        return badRequestResult(
          "'completed' status is set automatically when the trip ends",
        );
      }
      return tripClient.updateTrip(slug[0], body as UpdateTripInput, auth);
    };
  }
  // PATCH /trips/:id/itinerary/:item_id
  if (slug.length === 3 && slug[1] === "itinerary") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return tripClient.updateItinerary(
        slug[0],
        slug[2],
        body as UpdateItineraryInput,
        auth,
      );
    };
  }
  // PATCH /trips/:id/todos/:todo_id
  if (slug.length === 3 && slug[1] === "todos") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return tripClient.updateTodo(
        slug[0],
        slug[2],
        body as UpdateTodoInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /trips/:id
  if (slug.length === 1) {
    return (_req, auth) => tripClient.deleteTrip(slug[0], auth);
  }
  // DELETE /trips/:id/room/members/:user_id
  if (
    slug.length === 4 &&
    slug[1] === "room" &&
    slug[2] === "members"
  ) {
    return (_req, auth) => tripClient.removeMember(slug[0], slug[3], auth);
  }
  // DELETE /trips/:id/itinerary/:item_id
  if (slug.length === 3 && slug[1] === "itinerary") {
    return (_req, auth) => tripClient.deleteItinerary(slug[0], slug[2], auth);
  }
  // DELETE /trips/:id/todos/:todo_id
  if (slug.length === 3 && slug[1] === "todos") {
    return (_req, auth) => tripClient.deleteTodo(slug[0], slug[2], auth);
  }
  return null;
};

const readListTripsQuery = (req: NextRequest): ListTripsQueryInput => {
  const url = new URL(req.url);
  const out: ListTripsQueryInput = {};
  const status = url.searchParams.get("status");
  if (status === "planning" || status === "ongoing" || status === "completed") {
    out.status = status;
  }
  const q = url.searchParams.get("q");
  if (q) out.q = q;
  const role = url.searchParams.get("role");
  if (role === "owner" || role === "member" || role === "all") {
    out.role = role;
  }
  const limit = url.searchParams.get("limit");
  if (limit) {
    const n = Number(limit);
    if (Number.isFinite(n) && n > 0) out.limit = n;
  }
  const offset = url.searchParams.get("offset");
  if (offset) {
    const n = Number(offset);
    if (Number.isFinite(n) && n >= 0) out.offset = n;
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
      message: `unknown trips op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );

const isCompletedStatusInPayload = (body: unknown): boolean => {
  if (!body || typeof body !== "object") return false;
  const status = (body as { status?: unknown }).status;
  return typeof status === "string" && status.toLowerCase() === "completed";
};

const badRequestResult = (message: string): ApiResult<unknown> => ({
  ok: false,
  status: 400,
  code: 400,
  message,
  error: {
    category: "validation",
    status: 400,
    code: 400,
    message,
    retryable: false,
  },
  payload: { code: 400, message },
  raw: "",
});
