import "server-only";
import { NextRequest, NextResponse } from "next/server";

import type {
  AlbumCompletePhotoInput,
  AlbumCreateShareInput,
  AlbumInitUploadInput,
  AlbumUpdatePhotoInput,
} from "../../_lib/clients/album-client";
import { albumClient } from "../../_lib/clients/album-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (req: NextRequest, auth: AuthArg) => Promise<NextResponse>;

export async function GET(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPost(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function PATCH(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPatch(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /album/trips/:tripID/photos
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "photos") {
    return async (_req, auth) =>
      toResponse(await albumClient.listPhotos(slug[1], auth));
  }
  // GET /album/trips/:tripID/shares
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "shares") {
    return async (_req, auth) =>
      toResponse(await albumClient.listShareTokens(slug[1], auth));
  }
  // GET /album/photos/:photoID/download
  if (slug.length === 3 && slug[0] === "photos" && slug[2] === "download") {
    return async (_req, auth) =>
      toResponse(await albumClient.downloadOriginal(slug[1], auth));
  }
  // GET /album/public/:token — no auth
  if (slug.length === 2 && slug[0] === "public") {
    return async (_req, auth) =>
      toResponse(await albumClient.resolvePublicAlbum(slug[1], auth));
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /album/trips/:tripID/init-upload
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "init-upload") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await albumClient.initUpload(
          slug[1],
          body as AlbumInitUploadInput,
          auth,
        ),
      );
    };
  }
  // POST /album/trips/:tripID/photos
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "photos") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await albumClient.completePhoto(
          slug[1],
          body as AlbumCompletePhotoInput,
          auth,
        ),
      );
    };
  }
  // POST /album/trips/:tripID/shares
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "shares") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await albumClient.createShareToken(
          slug[1],
          body as AlbumCreateShareInput | null,
          auth,
        ),
      );
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /album/photos/:photoID
  if (slug.length === 2 && slug[0] === "photos") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await albumClient.updatePhoto(
          slug[1],
          body as AlbumUpdatePhotoInput,
          auth,
        ),
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /album/photos/:photoID
  if (slug.length === 2 && slug[0] === "photos") {
    return async (_req, auth) =>
      toResponse(await albumClient.deletePhoto(slug[1], auth));
  }
  // DELETE /album/shares/:tokenID
  if (slug.length === 2 && slug[0] === "shares") {
    return async (_req, auth) =>
      toResponse(await albumClient.revokeShareToken(slug[1], auth));
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

const notFound = (method: string, slug: string[]): NextResponse =>
  NextResponse.json(
    {
      code: 404,
      message: `unknown album op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
