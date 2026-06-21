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
  ItineraryView,
  JoinRoomView,
  RoomPreviewView,
  RoomView,
  TodoView,
  TripView,
  toItineraryView,
  toJoinRoomView,
  toRoomPreviewView,
  toRoomView,
  toTodoView,
  toTripView,
} from "../contracts/frontend";
import {
  UpstreamCreateItineraryPayload,
  UpstreamCreateTodoPayload,
  UpstreamCreateTripPayload,
  UpstreamItineraryResponse,
  UpstreamJoinRoomPayload,
  UpstreamJoinRoomResponse,
  UpstreamListTripsQuery,
  UpstreamRoomPreview,
  UpstreamRoomResponse,
  UpstreamTodoResponse,
  UpstreamTripResponse,
  UpstreamUpdateItineraryPayload,
  UpstreamUpdateTodoPayload,
  UpstreamUpdateTripPayload,
} from "../contracts/upstream";

export type CreateTripInput = UpstreamCreateTripPayload;
export type UpdateTripInput = UpstreamUpdateTripPayload;
export type ListTripsQueryInput = UpstreamListTripsQuery;
export type JoinRoomInput = UpstreamJoinRoomPayload;
export type CreateItineraryInput = UpstreamCreateItineraryPayload;
export type UpdateItineraryInput = UpstreamUpdateItineraryPayload;
export type CreateTodoInput = UpstreamCreateTodoPayload;
export type UpdateTodoInput = UpstreamUpdateTodoPayload;

export type {
  ItineraryView,
  JoinRoomView,
  RoomPreviewView,
  RoomView,
  TodoView,
  TripView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class TripClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  // ---- Trip ----

  async listTrips(
    auth: AuthCtx,
    query: ListTripsQueryInput = {},
  ): Promise<ApiResult<TripView[]>> {
    const qs = buildListTripsQuery(query);
    const path = qs ? `/trips?${qs}` : "/trips";
    const res = await this.callProtected<UpstreamTripResponse[]>(auth, (opts) =>
      this.http.get<UpstreamTripResponse[]>(path, opts),
    );
    return mapData(res, (rows) => rows.map(toTripView));
  }

  async createTrip(
    input: CreateTripInput,
    auth: AuthCtx,
  ): Promise<ApiResult<TripView>> {
    const res = await this.callProtected<UpstreamTripResponse>(auth, (opts) =>
      this.http.post<UpstreamTripResponse>("/trips", input, opts),
    );
    return mapData(res, toTripView);
  }

  async getTrip(tripID: string, auth: AuthCtx): Promise<ApiResult<TripView>> {
    const res = await this.callProtected<UpstreamTripResponse>(auth, (opts) =>
      this.http.get<UpstreamTripResponse>(
        `/trips/${encodeURIComponent(tripID)}`,
        opts,
      ),
    );
    return mapData(res, toTripView);
  }

  async updateTrip(
    tripID: string,
    input: UpdateTripInput,
    auth: AuthCtx,
  ): Promise<ApiResult<TripView>> {
    const res = await this.callProtected<UpstreamTripResponse>(auth, (opts) =>
      this.http.patch<UpstreamTripResponse>(
        `/trips/${encodeURIComponent(tripID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toTripView);
  }

  async deleteTrip(tripID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(`/trips/${encodeURIComponent(tripID)}`, opts),
    );
  }

  // ---- Room ----

  async getRoom(tripID: string, auth: AuthCtx): Promise<ApiResult<RoomView>> {
    const res = await this.callProtected<UpstreamRoomResponse>(auth, (opts) =>
      this.http.get<UpstreamRoomResponse>(
        `/trips/${encodeURIComponent(tripID)}/room`,
        opts,
      ),
    );
    return mapData(res, toRoomView);
  }

  async leaveRoom(tripID: string, auth: AuthCtx): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/trips/${encodeURIComponent(tripID)}/room/leave`,
        null,
        opts,
      ),
    );
  }

  async removeMember(
    tripID: string,
    userID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/trips/${encodeURIComponent(tripID)}/room/members/${encodeURIComponent(userID)}`,
        opts,
      ),
    );
  }

  async regenerateCode(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<RoomView>> {
    const res = await this.callProtected<UpstreamRoomResponse>(auth, (opts) =>
      this.http.post<UpstreamRoomResponse>(
        `/trips/${encodeURIComponent(tripID)}/room/code`,
        null,
        opts,
      ),
    );
    return mapData(res, toRoomView);
  }

  async previewRoomByCode(
    code: string,
    auth: AuthCtx,
  ): Promise<ApiResult<RoomPreviewView>> {
    const res = await this.callProtected<UpstreamRoomPreview>(auth, (opts) =>
      this.http.get<UpstreamRoomPreview>(
        `/rooms/by-code/${encodeURIComponent(code)}`,
        opts,
      ),
    );
    return mapData(res, toRoomPreviewView);
  }

  async joinRoom(
    input: JoinRoomInput,
    auth: AuthCtx,
  ): Promise<ApiResult<JoinRoomView>> {
    const res = await this.callProtected<UpstreamJoinRoomResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamJoinRoomResponse>("/rooms/join", input, opts),
    );
    return mapData(res, toJoinRoomView);
  }

  // ---- Itinerary ----

  async listItinerary(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ItineraryView[]>> {
    const res = await this.callProtected<UpstreamItineraryResponse[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamItineraryResponse[]>(
          `/trips/${encodeURIComponent(tripID)}/itinerary`,
          opts,
        ),
    );
    return mapData(res, (rows) => rows.map(toItineraryView));
  }

  async createItinerary(
    tripID: string,
    input: CreateItineraryInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ItineraryView>> {
    const res = await this.callProtected<UpstreamItineraryResponse>(
      auth,
      (opts) =>
        this.http.post<UpstreamItineraryResponse>(
          `/trips/${encodeURIComponent(tripID)}/itinerary`,
          input,
          opts,
        ),
    );
    return mapData(res, toItineraryView);
  }

  async updateItinerary(
    tripID: string,
    itemID: string,
    input: UpdateItineraryInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ItineraryView>> {
    const res = await this.callProtected<UpstreamItineraryResponse>(
      auth,
      (opts) =>
        this.http.patch<UpstreamItineraryResponse>(
          `/trips/${encodeURIComponent(tripID)}/itinerary/${encodeURIComponent(itemID)}`,
          input,
          opts,
        ),
    );
    return mapData(res, toItineraryView);
  }

  async deleteItinerary(
    tripID: string,
    itemID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/trips/${encodeURIComponent(tripID)}/itinerary/${encodeURIComponent(itemID)}`,
        opts,
      ),
    );
  }

  // ---- Todos ----

  async listTodos(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<TodoView[]>> {
    const res = await this.callProtected<UpstreamTodoResponse[]>(auth, (opts) =>
      this.http.get<UpstreamTodoResponse[]>(
        `/trips/${encodeURIComponent(tripID)}/todos`,
        opts,
      ),
    );
    return mapData(res, (rows) => rows.map(toTodoView));
  }

  async createTodo(
    tripID: string,
    input: CreateTodoInput,
    auth: AuthCtx,
  ): Promise<ApiResult<TodoView>> {
    const res = await this.callProtected<UpstreamTodoResponse>(auth, (opts) =>
      this.http.post<UpstreamTodoResponse>(
        `/trips/${encodeURIComponent(tripID)}/todos`,
        input,
        opts,
      ),
    );
    return mapData(res, toTodoView);
  }

  async updateTodo(
    tripID: string,
    todoID: string,
    input: UpdateTodoInput,
    auth: AuthCtx,
  ): Promise<ApiResult<TodoView>> {
    const res = await this.callProtected<UpstreamTodoResponse>(auth, (opts) =>
      this.http.patch<UpstreamTodoResponse>(
        `/trips/${encodeURIComponent(tripID)}/todos/${encodeURIComponent(todoID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toTodoView);
  }

  async deleteTodo(
    tripID: string,
    todoID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/trips/${encodeURIComponent(tripID)}/todos/${encodeURIComponent(todoID)}`,
        opts,
      ),
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

const buildListTripsQuery = (q: ListTripsQueryInput): string => {
  const qs = new URLSearchParams();
  if (q.status) qs.set("status", q.status);
  if (q.q) qs.set("q", q.q);
  if (q.role) qs.set("role", q.role);
  if (q.limit !== undefined) qs.set("limit", String(q.limit));
  if (q.offset !== undefined) qs.set("offset", String(q.offset));
  const s = qs.toString();
  return s;
};

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

export const tripClient = new TripClient();
