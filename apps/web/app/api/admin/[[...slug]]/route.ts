import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { adminClient } from "../../_lib/clients/admin-client";
import type {
  ElevateInput,
  ListAdminPostsQuery,
  ListAdminReportsQuery,
  ListAdminUsersQuery,
  ResolveReportInput,
} from "../../_lib/clients/admin-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";
import type {
  UpstreamAdminPostStatus,
  UpstreamAdminReportStatus,
  UpstreamAdminSubjectType,
  UpstreamAdminUserStatus,
} from "../../_lib/contracts/upstream";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (
  req: NextRequest,
  auth: AuthArg,
) => Promise<ApiResult<unknown>>;

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
  // GET /admin/users
  if (slug.length === 1 && slug[0] === "users") {
    return (req, auth) => adminClient.listUsers(auth, readListUsersQuery(req));
  }
  // GET /admin/users/:id
  if (slug.length === 2 && slug[0] === "users") {
    return (_req, auth) => adminClient.getUser(slug[1], auth);
  }
  // GET /admin/posts
  if (slug.length === 1 && slug[0] === "posts") {
    return (req, auth) => adminClient.listPosts(auth, readListPostsQuery(req));
  }
  // GET /admin/reports
  if (slug.length === 1 && slug[0] === "reports") {
    return (req, auth) =>
      adminClient.listReports(auth, readListReportsQuery(req));
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /admin/auth/elevate
  if (slug.length === 2 && slug[0] === "auth" && slug[1] === "elevate") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return adminClient.elevate((body ?? {}) as ElevateInput, auth);
    };
  }
  // POST /admin/auth/logout
  if (slug.length === 2 && slug[0] === "auth" && slug[1] === "logout") {
    return (_req, auth) => adminClient.adminLogout(auth);
  }
  // POST /admin/users/:id/block
  if (slug.length === 3 && slug[0] === "users" && slug[2] === "block") {
    return (_req, auth) => adminClient.blockUser(slug[1], auth);
  }
  // POST /admin/users/:id/unblock
  if (slug.length === 3 && slug[0] === "users" && slug[2] === "unblock") {
    return (_req, auth) => adminClient.unblockUser(slug[1], auth);
  }
  // POST /admin/users/:id/deactivate
  if (slug.length === 3 && slug[0] === "users" && slug[2] === "deactivate") {
    return (_req, auth) => adminClient.deactivateUser(slug[1], auth);
  }
  // POST /admin/posts/:id/restore
  if (slug.length === 3 && slug[0] === "posts" && slug[2] === "restore") {
    return (_req, auth) => adminClient.restorePost(slug[1], auth);
  }
  // POST /admin/reports/:id/resolve
  if (slug.length === 3 && slug[0] === "reports" && slug[2] === "resolve") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return adminClient.resolveReport(
        slug[1],
        (body ?? {}) as ResolveReportInput,
        auth,
      );
    };
  }
  // POST /admin/reports/:id/dismiss
  if (slug.length === 3 && slug[0] === "reports" && slug[2] === "dismiss") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return adminClient.dismissReport(
        slug[1],
        (body ?? {}) as ResolveReportInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /admin/posts/:id
  if (slug.length === 2 && slug[0] === "posts") {
    return (_req, auth) => adminClient.deletePost(slug[1], auth);
  }
  // DELETE /admin/comments/:id
  if (slug.length === 2 && slug[0] === "comments") {
    return (_req, auth) => adminClient.deleteComment(slug[1], auth);
  }
  return null;
};

const USER_STATUSES: UpstreamAdminUserStatus[] = [
  "active",
  "deactivated",
  "deleted",
];
const POST_STATUSES: UpstreamAdminPostStatus[] = [
  "draft",
  "published",
  "archived",
];
const REPORT_STATUSES: UpstreamAdminReportStatus[] = [
  "pending",
  "resolved",
  "dismissed",
];
const SUBJECT_TYPES: UpstreamAdminSubjectType[] = ["post", "comment", "user"];

const readListUsersQuery = (req: NextRequest): ListAdminUsersQuery => {
  const url = new URL(req.url);
  const out: ListAdminUsersQuery = {};
  const q = url.searchParams.get("q");
  if (q) out.q = q;
  const status = url.searchParams.get("status");
  if (status && (USER_STATUSES as string[]).includes(status)) {
    out.status = status as UpstreamAdminUserStatus;
  }
  const blocked = url.searchParams.get("is_blocked");
  if (blocked === "true") out.is_blocked = true;
  else if (blocked === "false") out.is_blocked = false;
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limit = parseLimit(url);
  if (limit !== undefined) out.limit = limit;
  return out;
};

const readListPostsQuery = (req: NextRequest): ListAdminPostsQuery => {
  const url = new URL(req.url);
  const out: ListAdminPostsQuery = {};
  const q = url.searchParams.get("q");
  if (q) out.q = q;
  const status = url.searchParams.get("status");
  if (status && (POST_STATUSES as string[]).includes(status)) {
    out.status = status as UpstreamAdminPostStatus;
  }
  const author = url.searchParams.get("author_id");
  if (author) out.author_id = author;
  const includeDel = url.searchParams.get("include_deleted");
  if (includeDel === "true") out.include_deleted = true;
  else if (includeDel === "false") out.include_deleted = false;
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limit = parseLimit(url);
  if (limit !== undefined) out.limit = limit;
  return out;
};

const readListReportsQuery = (req: NextRequest): ListAdminReportsQuery => {
  const url = new URL(req.url);
  const out: ListAdminReportsQuery = {};
  const status = url.searchParams.get("status");
  if (status && (REPORT_STATUSES as string[]).includes(status)) {
    out.status = status as UpstreamAdminReportStatus;
  }
  const kind = url.searchParams.get("subject_type");
  if (kind && (SUBJECT_TYPES as string[]).includes(kind)) {
    out.subject_type = kind as UpstreamAdminSubjectType;
  }
  const cursor = url.searchParams.get("cursor");
  if (cursor) out.cursor = cursor;
  const limit = parseLimit(url);
  if (limit !== undefined) out.limit = limit;
  return out;
};

const parseLimit = (url: URL): number | undefined => {
  const raw = url.searchParams.get("limit");
  if (!raw) return undefined;
  const n = Number(raw);
  if (Number.isFinite(n) && n > 0) return n;
  return undefined;
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
      message: `unknown admin op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
