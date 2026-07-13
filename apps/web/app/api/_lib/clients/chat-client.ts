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
  ChatMessageView,
  ChatReadReceiptView,
  ChatRoomView,
  ChatTicketView,
  toChatMessageView,
  toChatReadReceiptView,
  toChatRoomView,
  toChatTicketView,
} from "../contracts/frontend";
import {
  UpstreamChatMessage,
  UpstreamChatReadReceipt,
  UpstreamChatRoom,
  UpstreamChatTicketResponse,
  UpstreamEnsureDMPayload,
  UpstreamListChatMessagesQuery,
  UpstreamMarkReadPayload,
} from "../contracts/upstream";

export type EnsureDMInput = UpstreamEnsureDMPayload;
export type MarkReadInput = UpstreamMarkReadPayload;
export type ListMessagesQuery = UpstreamListChatMessagesQuery;

export type {
  ChatMessageTypeView,
  ChatMessageView,
  ChatReadReceiptView,
  ChatRoomTypeView,
  ChatRoomView,
  ChatTicketView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

// ChatClient proxies room + message operations for the Go chat service.
// The BFF forwards the user's session (access-token + CSRF) upstream so the
// browser never holds the JWT — same posture as MediaClient/RealtimeClient.
export class ChatClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listRooms(auth: AuthCtx): Promise<ApiResult<ChatRoomView[]>> {
    const res = await this.callProtected<UpstreamChatRoom[]>(auth, (opts) =>
      this.http.get<UpstreamChatRoom[]>("/chat/rooms", opts),
    );
    return mapData(res, (rooms) => (rooms ?? []).map(toChatRoomView));
  }

  async ensureDM(
    input: EnsureDMInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatRoomView>> {
    const res = await this.callProtected<UpstreamChatRoom>(auth, (opts) =>
      this.http.post<UpstreamChatRoom>("/chat/rooms/dm", input, opts),
    );
    return mapData(res, toChatRoomView);
  }

  async ensureTripRoom(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatRoomView>> {
    const res = await this.callProtected<UpstreamChatRoom>(auth, (opts) =>
      this.http.post<UpstreamChatRoom>(
        `/chat/trips/${encodeURIComponent(tripID)}/room`,
        null,
        opts,
      ),
    );
    return mapData(res, toChatRoomView);
  }

  async listMessages(
    roomID: string,
    query: ListMessagesQuery,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatMessageView[]>> {
    const qs = buildQuery(query);
    const suffix = qs ? `?${qs}` : "";
    const res = await this.callProtected<UpstreamChatMessage[]>(auth, (opts) =>
      this.http.get<UpstreamChatMessage[]>(
        `/chat/rooms/${encodeURIComponent(roomID)}/messages${suffix}`,
        opts,
      ),
    );
    return mapData(res, (msgs) => (msgs ?? []).map(toChatMessageView));
  }

  async listReadReceipts(
    roomID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatReadReceiptView[]>> {
    const res = await this.callProtected<UpstreamChatReadReceipt[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamChatReadReceipt[]>(
          `/chat/rooms/${encodeURIComponent(roomID)}/receipts`,
          opts,
        ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toChatReadReceiptView));
  }

  async markRead(
    roomID: string,
    input: MarkReadInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatReadReceiptView>> {
    const res = await this.callProtected<UpstreamChatReadReceipt>(
      auth,
      (opts) =>
        this.http.post<UpstreamChatReadReceipt>(
          `/chat/rooms/${encodeURIComponent(roomID)}/read`,
          input,
          opts,
        ),
    );
    return mapData(res, toChatReadReceiptView);
  }

  async hideRoom(
    roomID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/chat/rooms/${encodeURIComponent(roomID)}`,
        opts,
      ),
    );
  }

  async issueRoomTicket(
    roomID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ChatTicketView>> {
    const res = await this.callProtected<UpstreamChatTicketResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamChatTicketResponse>(
          `/chat/rooms/${encodeURIComponent(roomID)}/ticket`,
          null,
          opts,
        ),
    );
    return mapData(res, toChatTicketView);
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

const buildQuery = (query: ListMessagesQuery): string => {
  const params = new URLSearchParams();
  if (query.before) params.set("before", query.before);
  if (typeof query.limit === "number" && Number.isFinite(query.limit)) {
    params.set("limit", String(query.limit));
  }
  return params.toString();
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

export const chatClient = new ChatClient();
