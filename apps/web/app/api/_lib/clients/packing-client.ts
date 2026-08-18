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
import { PackingItemView, toPackingItemView } from "../contracts/frontend";
import {
  UpstreamCreatePackingItemPayload,
  UpstreamPackingItem,
  UpstreamUpdatePackingItemPayload,
} from "../contracts/upstream";

export type CreatePackingItemInput = UpstreamCreatePackingItemPayload;
export type UpdatePackingItemInput = UpstreamUpdatePackingItemPayload;

export type { PackingItemView } from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class PackingClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listItems(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PackingItemView[]>> {
    const res = await this.callProtected<UpstreamPackingItem[]>(auth, (opts) =>
      this.http.get<UpstreamPackingItem[]>(
        `/trips/${encodeURIComponent(tripID)}/packing/items`,
        opts,
      ),
    );
    return mapData(res, (items) => (items ?? []).map(toPackingItemView));
  }

  async createItem(
    tripID: string,
    input: CreatePackingItemInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PackingItemView>> {
    const res = await this.callProtected<UpstreamPackingItem>(auth, (opts) =>
      this.http.post<UpstreamPackingItem>(
        `/trips/${encodeURIComponent(tripID)}/packing/items`,
        input,
        opts,
      ),
    );
    return mapData(res, toPackingItemView);
  }

  async updateItem(
    itemID: string,
    input: UpdatePackingItemInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PackingItemView>> {
    const res = await this.callProtected<UpstreamPackingItem>(auth, (opts) =>
      this.http.patch<UpstreamPackingItem>(
        `/packing/items/${encodeURIComponent(itemID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toPackingItemView);
  }

  async deleteItem(itemID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/packing/items/${encodeURIComponent(itemID)}`,
        opts,
      ),
    );
  }

  async packItem(
    itemID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PackingItemView>> {
    const res = await this.callProtected<UpstreamPackingItem>(auth, (opts) =>
      this.http.post<UpstreamPackingItem>(
        `/packing/items/${encodeURIComponent(itemID)}/pack`,
        {},
        opts,
      ),
    );
    return mapData(res, toPackingItemView);
  }

  async unpackItem(
    itemID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PackingItemView>> {
    const res = await this.callProtected<UpstreamPackingItem>(auth, (opts) =>
      this.http.delete<UpstreamPackingItem>(
        `/packing/items/${encodeURIComponent(itemID)}/pack`,
        opts,
      ),
    );
    return mapData(res, toPackingItemView);
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

export const packingClient = new PackingClient();
