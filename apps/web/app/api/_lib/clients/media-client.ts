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
  AssetView,
  InitUploadView,
  PresignedGetView,
  toAssetView,
  toInitUploadView,
  toPresignedGetView,
} from "../contracts/frontend";
import {
  UpstreamAssetResponse,
  UpstreamCompleteUploadPayload,
  UpstreamInitUploadPayload,
  UpstreamInitUploadResponse,
  UpstreamPresignedGetResponse,
} from "../contracts/upstream";

export type InitUploadInput = UpstreamInitUploadPayload;
export type CompleteUploadInput = UpstreamCompleteUploadPayload;

export type {
  AssetView,
  InitUploadView,
  MediaPurposeView,
  PresignedGetView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

// MediaClient proxies the two-phase upload handshake and asset lookups.
// The browser PUTs the file body directly to the presigned URL returned by
// initUpload; the BFF never sees the bytes.
export class MediaClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async initUpload(
    input: InitUploadInput,
    auth: AuthCtx,
  ): Promise<ApiResult<InitUploadView>> {
    const res = await this.callProtected<UpstreamInitUploadResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamInitUploadResponse>(
          "/media/upload/init",
          input,
          opts,
        ),
    );
    return mapData(res, toInitUploadView);
  }

  async completeUpload(
    input: CompleteUploadInput,
    auth: AuthCtx,
  ): Promise<ApiResult<AssetView>> {
    const res = await this.callProtected<UpstreamAssetResponse>(auth, (opts) =>
      this.http.post<UpstreamAssetResponse>(
        "/media/upload/complete",
        input,
        opts,
      ),
    );
    return mapData(res, toAssetView);
  }

  async getAsset(
    assetID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<AssetView>> {
    const res = await this.callProtected<UpstreamAssetResponse>(auth, (opts) =>
      this.http.get<UpstreamAssetResponse>(
        `/media/${encodeURIComponent(assetID)}`,
        opts,
      ),
    );
    return mapData(res, toAssetView);
  }

  async presignRead(
    assetID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PresignedGetView>> {
    const res = await this.callProtected<UpstreamPresignedGetResponse>(
      auth,
      (opts) =>
        this.http.get<UpstreamPresignedGetResponse>(
          `/media/${encodeURIComponent(assetID)}/url`,
          opts,
        ),
    );
    return mapData(res, toPresignedGetView);
  }

  async softDelete(
    assetID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/media/${encodeURIComponent(assetID)}`, opts),
    );
  }

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

export const mediaClient = new MediaClient();
