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
  CalendarEventView,
  toCalendarEventView,
} from "../contracts/frontend";
import {
  UpstreamCreateEventPayload,
  UpstreamEventResponse,
  UpstreamListEventsQuery,
  UpstreamUpdateEventPayload,
} from "../contracts/upstream";

export type CreateEventInput = UpstreamCreateEventPayload;
export type UpdateEventInput = UpstreamUpdateEventPayload;
export type ListEventsQueryInput = UpstreamListEventsQuery;

export type { CalendarEventView } from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class CalendarClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  async listMyEvents(
    auth: AuthCtx,
    query: ListEventsQueryInput = {},
  ): Promise<ApiResult<CalendarEventView[]>> {
    const qs = buildRangeQuery(query);
    const path = qs ? `/calendar/events?${qs}` : "/calendar/events";
    const res = await this.callProtected<UpstreamEventResponse[]>(
      auth,
      (opts) => this.http.get<UpstreamEventResponse[]>(path, opts),
    );
    return mapData(res, (rows) => rows.map(toCalendarEventView));
  }

  async createEvent(
    input: CreateEventInput,
    auth: AuthCtx,
  ): Promise<ApiResult<CalendarEventView>> {
    const res = await this.callProtected<UpstreamEventResponse>(auth, (opts) =>
      this.http.post<UpstreamEventResponse>("/calendar/events", input, opts),
    );
    return mapData(res, toCalendarEventView);
  }

  async getEvent(
    eventID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<CalendarEventView>> {
    const res = await this.callProtected<UpstreamEventResponse>(auth, (opts) =>
      this.http.get<UpstreamEventResponse>(
        `/calendar/events/${encodeURIComponent(eventID)}`,
        opts,
      ),
    );
    return mapData(res, toCalendarEventView);
  }

  async updateEvent(
    eventID: string,
    input: UpdateEventInput,
    auth: AuthCtx,
  ): Promise<ApiResult<CalendarEventView>> {
    const res = await this.callProtected<UpstreamEventResponse>(auth, (opts) =>
      this.http.patch<UpstreamEventResponse>(
        `/calendar/events/${encodeURIComponent(eventID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toCalendarEventView);
  }

  async deleteEvent(
    eventID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/calendar/events/${encodeURIComponent(eventID)}`,
        opts,
      ),
    );
  }

  async listTripEvents(
    tripID: string,
    auth: AuthCtx,
    query: ListEventsQueryInput = {},
  ): Promise<ApiResult<CalendarEventView[]>> {
    const qs = buildRangeQuery(query);
    const base = `/calendar/trips/${encodeURIComponent(tripID)}/events`;
    const path = qs ? `${base}?${qs}` : base;
    const res = await this.callProtected<UpstreamEventResponse[]>(
      auth,
      (opts) => this.http.get<UpstreamEventResponse[]>(path, opts),
    );
    return mapData(res, (rows) => rows.map(toCalendarEventView));
  }

  async listFriendEvents(
    friendID: string,
    auth: AuthCtx,
    query: ListEventsQueryInput = {},
  ): Promise<ApiResult<CalendarEventView[]>> {
    const qs = buildRangeQuery(query);
    const base = `/calendar/friends/${encodeURIComponent(friendID)}/events`;
    const path = qs ? `${base}?${qs}` : base;
    const res = await this.callProtected<UpstreamEventResponse[]>(
      auth,
      (opts) => this.http.get<UpstreamEventResponse[]>(path, opts),
    );
    return mapData(res, (rows) => rows.map(toCalendarEventView));
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

const buildRangeQuery = (q: ListEventsQueryInput): string => {
  const qs = new URLSearchParams();
  if (q.from) qs.set("from", q.from);
  if (q.to) qs.set("to", q.to);
  return qs.toString();
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

export const calendarClient = new CalendarClient();
