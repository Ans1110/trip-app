import "server-only";
import { NextRequest, NextResponse } from "next/server";

import type {
  AddPollOptionInput,
  CastVoteInput,
  CreatePollInput,
} from "../../_lib/clients/vote-client";
import { voteClient } from "../../_lib/clients/vote-client";
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

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return toResponse(await dispatch(req, buildAuth(req)));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /vote/trips/:tripID/polls
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "polls") {
    return (_req, auth) => voteClient.listPolls(slug[1], auth);
  }
  // GET /vote/polls/:pollID
  if (slug.length === 2 && slug[0] === "polls") {
    return (_req, auth) => voteClient.getPoll(slug[1], auth);
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /vote/trips/:tripID/polls
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "polls") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return voteClient.createPoll(slug[1], body as CreatePollInput, auth);
    };
  }
  // POST /vote/polls/:pollID/options
  if (slug.length === 3 && slug[0] === "polls" && slug[2] === "options") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return voteClient.addOption(slug[1], body as AddPollOptionInput, auth);
    };
  }
  // POST /vote/polls/:pollID/vote
  if (slug.length === 3 && slug[0] === "polls" && slug[2] === "vote") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return voteClient.castVote(slug[1], body as CastVoteInput, auth);
    };
  }
  // POST /vote/polls/:pollID/close
  if (slug.length === 3 && slug[0] === "polls" && slug[2] === "close") {
    return (_req, auth) => voteClient.closePoll(slug[1], auth);
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /vote/polls/options/:optionID — must precede /vote/polls/:pollID
  if (slug.length === 3 && slug[0] === "polls" && slug[1] === "options") {
    return (_req, auth) => voteClient.deleteOption(slug[2], auth);
  }
  // DELETE /vote/polls/:pollID
  if (slug.length === 2 && slug[0] === "polls") {
    return (_req, auth) => voteClient.deletePoll(slug[1], auth);
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
      message: `unknown vote op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
