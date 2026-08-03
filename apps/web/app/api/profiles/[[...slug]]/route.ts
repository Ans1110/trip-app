import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { profileClient } from "../../_lib/clients/profile-client";
import type {
  ProfilePageQueryInput,
  UpdateProfileInput,
} from "../../_lib/clients/profile-client";
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
  // GET /profiles/me
  if (slug.length === 1 && slug[0] === "me") {
    return (_req, auth) => profileClient.getMyProfile(auth);
  }
  // GET /profiles/recommendations?limit=
  if (slug.length === 1 && slug[0] === "recommendations") {
    return (req, auth) => {
      const url = new URL(req.url);
      const limitParam = url.searchParams.get("limit");
      const limit = limitParam ? Number(limitParam) : undefined;
      return profileClient.recommendations(auth, { limit });
    };
  }
  // GET /profiles/by-username/:username
  if (slug.length === 2 && slug[0] === "by-username") {
    return (_req, auth) => profileClient.getProfileByUsername(slug[1], auth);
  }
  // GET /profiles/:user_id/followers
  if (slug.length === 2 && slug[1] === "followers") {
    return (req, auth) =>
      profileClient.listFollowers(slug[0], auth, readPageQuery(req));
  }
  // GET /profiles/:user_id/following
  if (slug.length === 2 && slug[1] === "following") {
    return (req, auth) =>
      profileClient.listFollowing(slug[0], auth, readPageQuery(req));
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /profiles/:user_id/follow
  if (slug.length === 2 && slug[1] === "follow") {
    return (_req, auth) => profileClient.follow(slug[0], auth);
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /profiles/me
  if (slug.length === 1 && slug[0] === "me") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return profileClient.updateMyProfile(
        (body ?? {}) as UpdateProfileInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /profiles/:user_id/follow
  if (slug.length === 2 && slug[1] === "follow") {
    return (_req, auth) => profileClient.unfollow(slug[0], auth);
  }
  return null;
};

const readPageQuery = (req: NextRequest): ProfilePageQueryInput => {
  const url = new URL(req.url);
  const out: ProfilePageQueryInput = {};
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
      message: `unknown profiles op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
