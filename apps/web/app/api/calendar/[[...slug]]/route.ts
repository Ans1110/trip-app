import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { calendarClient } from "../../_lib/clients/calendar-client";
import type {
  CreateEventInput,
  ListEventsQueryInput,
  UpdateEventInput,
} from "../../_lib/clients/calendar-client";
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
  // GET /calendar/events?from=&to=
  if (slug.length === 1 && slug[0] === "events") {
    return (req, auth) => calendarClient.listMyEvents(auth, readRange(req));
  }
  // GET /calendar/events/:id
  if (slug.length === 2 && slug[0] === "events") {
    return (_req, auth) => calendarClient.getEvent(slug[1], auth);
  }
  // GET /calendar/trips/:trip_id/events?from=&to=
  if (
    slug.length === 3 &&
    slug[0] === "trips" &&
    slug[2] === "events"
  ) {
    return (req, auth) =>
      calendarClient.listTripEvents(slug[1], auth, readRange(req));
  }
  // GET /calendar/friends/:friend_id/events?from=&to=
  if (
    slug.length === 3 &&
    slug[0] === "friends" &&
    slug[2] === "events"
  ) {
    return (req, auth) =>
      calendarClient.listFriendEvents(slug[1], auth, readRange(req));
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /calendar/events
  if (slug.length === 1 && slug[0] === "events") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return calendarClient.createEvent(body as CreateEventInput, auth);
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /calendar/events/:id
  if (slug.length === 2 && slug[0] === "events") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return calendarClient.updateEvent(
        slug[1],
        body as UpdateEventInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /calendar/events/:id
  if (slug.length === 2 && slug[0] === "events") {
    return (_req, auth) => calendarClient.deleteEvent(slug[1], auth);
  }
  return null;
};

const readRange = (req: NextRequest): ListEventsQueryInput => {
  const url = new URL(req.url);
  const out: ListEventsQueryInput = {};
  const from = url.searchParams.get("from");
  if (from) out.from = from;
  const to = url.searchParams.get("to");
  if (to) out.to = to;
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
      message: `unknown calendar op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
