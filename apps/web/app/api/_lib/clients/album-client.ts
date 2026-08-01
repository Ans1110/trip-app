import "server-only";
import { circuitOpenError } from "../errors";
import {
  ApiErr,
  ApiResult,
  Envelope,
  httpClient,
  HttpClient,
  HttpOptions,
} from "../http-client";
import { RequestContext } from "../request-context";
import { ensureCsrfToken, readSessionTokens } from "../session-store";
import {
  AlbumCreateShareView,
  AlbumDownloadOriginalView,
  AlbumInitUploadView,
  AlbumPhotoWithURLsView,
  AlbumPublicView,
  AlbumShareTokenView,
  toAlbumCreateShareView,
  toAlbumDownloadOriginalView,
  toAlbumInitUploadView,
  toAlbumPhotoWithURLsView,
  toAlbumPublicView,
  toAlbumShareTokenView,
} from "../contracts/frontend";
import {
  UpstreamAlbumCompletePhotoPayload,
  UpstreamAlbumCreateShareResponse,
  UpstreamAlbumCreateSharePayload,
  UpstreamAlbumDownloadOriginal,
  UpstreamAlbumInitUploadPayload,
  UpstreamAlbumInitUploadResponse,
  UpstreamAlbumPhotoWithURLs,
  UpstreamAlbumPublicResponse,
  UpstreamAlbumShareToken,
  UpstreamAlbumUpdatePhotoPayload,
} from "../contracts/upstream";

export type AlbumInitUploadInput = UpstreamAlbumInitUploadPayload;
export type AlbumCompletePhotoInput = UpstreamAlbumCompletePhotoPayload;
export type AlbumUpdatePhotoInput = UpstreamAlbumUpdatePhotoPayload;
export type AlbumCreateShareInput = UpstreamAlbumCreateSharePayload;

export type {
  AlbumCreateShareView,
  AlbumDownloadOriginalView,
  AlbumInitUploadSlotView,
  AlbumInitUploadView,
  AlbumPhotoView,
  AlbumPhotoWithURLsView,
  AlbumPublicView,
  AlbumShareTokenView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class AlbumClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  // ---- trip-scoped ----

  async initUpload(
    tripID: string,
    input: AlbumInitUploadInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumInitUploadView>> {
    const res = await this.callProtected<UpstreamAlbumInitUploadResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamAlbumInitUploadResponse>(
          `/trips/${encodeURIComponent(tripID)}/album/init-upload`,
          input,
          opts,
        ),
    );
    return mapData(res, toAlbumInitUploadView);
  }

  async completePhoto(
    tripID: string,
    input: AlbumCompletePhotoInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumPhotoWithURLsView>> {
    const res = await this.callProtected<UpstreamAlbumPhotoWithURLs>(
      auth,
      (opts) =>
        this.http.post<UpstreamAlbumPhotoWithURLs>(
          `/trips/${encodeURIComponent(tripID)}/album/photos`,
          input,
          opts,
        ),
    );
    return mapData(res, toAlbumPhotoWithURLsView);
  }

  async listPhotos(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumPhotoWithURLsView[]>> {
    const res = await this.callProtected<UpstreamAlbumPhotoWithURLs[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamAlbumPhotoWithURLs[]>(
          `/trips/${encodeURIComponent(tripID)}/album/photos`,
          opts,
        ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toAlbumPhotoWithURLsView));
  }

  // ---- photo-scoped ----

  async updatePhoto(
    photoID: string,
    input: AlbumUpdatePhotoInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumPhotoWithURLsView>> {
    const res = await this.callProtected<UpstreamAlbumPhotoWithURLs>(
      auth,
      (opts) =>
        this.http.patch<UpstreamAlbumPhotoWithURLs>(
          `/albums/photos/${encodeURIComponent(photoID)}`,
          input,
          opts,
        ),
    );
    return mapData(res, toAlbumPhotoWithURLsView);
  }

  async deletePhoto(
    photoID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/albums/photos/${encodeURIComponent(photoID)}`,
        opts,
      ),
    );
  }

  async downloadOriginal(
    photoID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumDownloadOriginalView>> {
    const res = await this.callProtected<UpstreamAlbumDownloadOriginal>(
      auth,
      (opts) =>
        this.http.get<UpstreamAlbumDownloadOriginal>(
          `/albums/photos/${encodeURIComponent(photoID)}/download`,
          opts,
        ),
    );
    return mapData(res, toAlbumDownloadOriginalView);
  }

  // ---- share tokens ----

  async createShareToken(
    tripID: string,
    input: AlbumCreateShareInput | null,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumCreateShareView>> {
    const res = await this.callProtected<UpstreamAlbumCreateShareResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamAlbumCreateShareResponse>(
          `/trips/${encodeURIComponent(tripID)}/album/shares`,
          input,
          opts,
        ),
    );
    return mapData(res, toAlbumCreateShareView);
  }

  async listShareTokens(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumShareTokenView[]>> {
    const res = await this.callProtected<UpstreamAlbumShareToken[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamAlbumShareToken[]>(
          `/trips/${encodeURIComponent(tripID)}/album/shares`,
          opts,
        ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toAlbumShareTokenView));
  }

  async revokeShareToken(
    tokenID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/albums/shares/${encodeURIComponent(tokenID)}`,
        opts,
      ),
    );
  }

  // No session, no CSRF: the token in the URL is the credential.
  async resolvePublicAlbum(
    token: string,
    auth: AuthCtx,
  ): Promise<ApiResult<AlbumPublicView>> {
    const res = await this.http.get<UpstreamAlbumPublicResponse>(
      `/public/albums/${encodeURIComponent(token)}`,
      { ctx: auth.ctx, signal: auth.signal },
    );
    return mapData(res, toAlbumPublicView);
  }

  // ---- plumbing ----

  private protectedOpts(auth: AuthCtx, deps: ProtectedDeps): HttpOptions {
    return {
      ctx: auth.ctx,
      signal: auth.signal,
      accessToken: deps.accessToken,
      csrfToken: deps.csrfToken,
    };
  }

  private async resolveProtected(
    auth: AuthCtx,
  ): Promise<{ deps: ProtectedDeps } | { error: ApiErr }> {
    const tokens = await readSessionTokens();
    if (!tokens.accessToken) return { error: MISSING_SESSION };
    const csrf = tokens.csrfToken ?? (await ensureCsrfToken(auth.ctx)) ?? null;
    if (!csrf) return { error: MISSING_CSRF };
    return {
      deps: {
        accessToken: tokens.accessToken,
        csrfToken: csrf,
      },
    };
  }

  private async callProtected<T>(
    auth: AuthCtx,
    invoke: (opts: HttpOptions) => Promise<ApiResult<T>>,
  ): Promise<ApiResult<T>> {
    const resolved = await this.resolveProtected(auth);
    if ("error" in resolved) return resolved.error as ApiResult<T>;
    return invoke(this.protectedOpts(auth, resolved.deps));
  }
}

const mapData = <TIn, TOut>(
  result: ApiResult<TIn>,
  map: (v: TIn) => TOut,
): ApiResult<TOut> => {
  if (!result.ok) return result;
  return {
    ...result,
    data: map(result.data),
    envelope: result.envelope as unknown as Envelope<TOut> | undefined,
  };
};

function unauthorizedError(message: string): ApiErr {
  return {
    ok: false,
    status: 401,
    code: 401,
    message,
    error: {
      category: "auth",
      status: 401,
      code: 401,
      message,
      retryable: false,
    },
    payload: null,
    raw: "",
  };
}

function circuitGuardError(message: string): ApiErr {
  const base = circuitOpenError();
  return {
    ok: false,
    status: base.status,
    code: base.code,
    message,
    error: { ...base, message },
    payload: null,
    raw: "",
  };
}

export const albumClient = new AlbumClient();
