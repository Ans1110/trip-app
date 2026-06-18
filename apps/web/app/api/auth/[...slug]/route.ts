import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { authClient } from "../../_lib/clients/auth-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug: string[] }> };

type Op = (req: NextRequest, body: unknown, auth: AuthArg) => Promise<ApiResult<unknown>>;
type AuthArg = { ctx: ReturnType<typeof extractRequestContext>; signal?: AbortSignal };

const POST_OPS: Record<string, Op> = {
  register: (_req, body, auth) => authClient.register(body as never, auth),
  login: (_req, body, auth) => authClient.login(body as never, auth),
  refresh: (_req, _body, auth) => authClient.refresh(auth),
  logout: (_req, _body, auth) => authClient.logout(auth),
  "forgot-password": (_req, body, auth) =>
    authClient.forgotPassword(body as never, auth),
  "reset-password": (_req, body, auth) =>
    authClient.resetPassword(body as never, auth),
  "verify-email": (_req, body, auth) =>
    authClient.verifyEmail(body as never, auth),
  "resend-verification": (_req, body, auth) =>
    authClient.resendVerification(body as never, auth),
};

const GET_OPS: Record<string, Op> = {
  me: (_req, _body, auth) => authClient.me(auth),
};

export async function POST(req: NextRequest, { params }: RouteParams) {
  const { slug } = await params;
  const op = POST_OPS[slug.join("/")];
  if (!op) return notFound(slug);

  const body = await readJson(req);
  if (body instanceof Response) return body;

  const auth = buildAuth(req);
  const result = await op(req, body, auth);
  return toResponse(result);
}

export async function GET(req: NextRequest, { params }: RouteParams) {
  const { slug } = await params;
  const op = GET_OPS[slug.join("/")];
  if (!op) return notFound(slug);

  const auth = buildAuth(req);
  const result = await op(req, undefined, auth);
  return toResponse(result);
}

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

const notFound = (slug: string[]): NextResponse =>
  NextResponse.json(
    { code: 404, message: `unknown auth op: ${slug.join("/")}` },
    { status: 404 },
  );
