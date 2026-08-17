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
  PlaceDetailsView,
  PlaceSummaryView,
  RouteView,
  SavedPlaceView,
  toPlaceDetailsView,
  toPlaceSummaryView,
  toRouteView,
  toSavedPlaceView,
  toTripMemberPositionView,
  toWeatherForecastView,
  TripMemberPositionView,
  WeatherForecastView,
} from "../contracts/frontend";
import {
  UpstreamComputeRoutePayload,
  UpstreamCreateSavedPlacePayload,
  UpstreamListSavedPlacesQuery,
  UpstreamNearbyPlacesQuery,
  UpstreamPlaceDetails,
  UpstreamPlaceSummary,
  UpstreamRoute,
  UpstreamSavedPlace,
  UpstreamSearchPlacesQuery,
  UpstreamShareTripPositionPayload,
  UpstreamTripMemberPosition,
  UpstreamUpdateSavedPlacePayload,
  UpstreamWeatherForecast,
  UpstreamWeatherQuery,
} from "../contracts/upstream";

export type CreateSavedPlaceInput = UpstreamCreateSavedPlacePayload;
export type UpdateSavedPlaceInput = UpstreamUpdateSavedPlacePayload;
export type ListSavedPlacesQueryInput = UpstreamListSavedPlacesQuery;
export type SearchPlacesQueryInput = UpstreamSearchPlacesQuery;
export type NearbyPlacesQueryInput = UpstreamNearbyPlacesQuery;
export type ComputeRouteInput = UpstreamComputeRoutePayload;
export type WeatherQueryInput = UpstreamWeatherQuery;
export type ShareTripPositionInput = UpstreamShareTripPositionPayload;

export type {
  PlaceDetailsView,
  PlaceSummaryView,
  RouteView,
  SavedPlaceView,
  TripMemberPositionView,
  WeatherForecastView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

export class LocationClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  // ---- Saved places ----

  async listSavedPlaces(
    auth: AuthCtx,
    query: ListSavedPlacesQueryInput = {},
  ): Promise<ApiResult<SavedPlaceView[]>> {
    const qs = buildSavedListQuery(query);
    const path = qs ? `/location/saved?${qs}` : "/location/saved";
    const res = await this.callProtected<UpstreamSavedPlace[]>(auth, (opts) =>
      this.http.get<UpstreamSavedPlace[]>(path, opts),
    );
    return mapData(res, (rows) => rows.map(toSavedPlaceView));
  }

  async createSavedPlace(
    input: CreateSavedPlaceInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SavedPlaceView>> {
    const res = await this.callProtected<UpstreamSavedPlace>(auth, (opts) =>
      this.http.post<UpstreamSavedPlace>("/location/saved", input, opts),
    );
    return mapData(res, toSavedPlaceView);
  }

  async getSavedPlace(
    placeID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<SavedPlaceView>> {
    const res = await this.callProtected<UpstreamSavedPlace>(auth, (opts) =>
      this.http.get<UpstreamSavedPlace>(
        `/location/saved/${encodeURIComponent(placeID)}`,
        opts,
      ),
    );
    return mapData(res, toSavedPlaceView);
  }

  async updateSavedPlace(
    placeID: string,
    input: UpdateSavedPlaceInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SavedPlaceView>> {
    const res = await this.callProtected<UpstreamSavedPlace>(auth, (opts) =>
      this.http.patch<UpstreamSavedPlace>(
        `/location/saved/${encodeURIComponent(placeID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toSavedPlaceView);
  }

  async deleteSavedPlace(
    placeID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/location/saved/${encodeURIComponent(placeID)}`,
        opts,
      ),
    );
  }

  // ---- Google Places passthroughs ----

  async searchPlaces(
    query: SearchPlacesQueryInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PlaceSummaryView[]>> {
    const qs = buildSearchQuery(query);
    const res = await this.callProtected<UpstreamPlaceSummary[]>(auth, (opts) =>
      this.http.get<UpstreamPlaceSummary[]>(
        `/location/places/search?${qs}`,
        opts,
      ),
    );
    return mapData(res, (rows) => rows.map(toPlaceSummaryView));
  }

  async nearbyPlaces(
    query: NearbyPlacesQueryInput,
    auth: AuthCtx,
  ): Promise<ApiResult<PlaceSummaryView[]>> {
    const qs = buildNearbyQuery(query);
    const res = await this.callProtected<UpstreamPlaceSummary[]>(auth, (opts) =>
      this.http.get<UpstreamPlaceSummary[]>(
        `/location/places/nearby?${qs}`,
        opts,
      ),
    );
    return mapData(res, (rows) => rows.map(toPlaceSummaryView));
  }

  async placeDetails(
    placeID: string,
    lang: string | undefined,
    auth: AuthCtx,
  ): Promise<ApiResult<PlaceDetailsView>> {
    const qs = lang ? `?lang=${encodeURIComponent(lang)}` : "";
    const res = await this.callProtected<UpstreamPlaceDetails>(auth, (opts) =>
      this.http.get<UpstreamPlaceDetails>(
        `/location/places/${encodeURIComponent(placeID)}${qs}`,
        opts,
      ),
    );
    return mapData(res, toPlaceDetailsView);
  }

  // ---- Routes ----

  async computeRoute(
    input: ComputeRouteInput,
    auth: AuthCtx,
  ): Promise<ApiResult<RouteView>> {
    const res = await this.callProtected<UpstreamRoute>(auth, (opts) =>
      this.http.post<UpstreamRoute>("/location/routes", input, opts),
    );
    return mapData(res, toRouteView);
  }

  // ---- Weather ----

  async weather(
    query: WeatherQueryInput,
    auth: AuthCtx,
  ): Promise<ApiResult<WeatherForecastView>> {
    const qs = buildWeatherQuery(query);
    const res = await this.callProtected<UpstreamWeatherForecast>(
      auth,
      (opts) =>
        this.http.get<UpstreamWeatherForecast>(`/location/weather?${qs}`, opts),
    );
    return mapData(res, toWeatherForecastView);
  }

  // ---- Trip member positions (opt-in live sharing) ----

  async shareTripPosition(
    tripID: string,
    input: ShareTripPositionInput,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.post<null>(
        `/location/trips/${encodeURIComponent(tripID)}/share`,
        input,
        opts,
      ),
    );
  }

  async revokeTripPosition(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/location/trips/${encodeURIComponent(tripID)}/share`,
        opts,
      ),
    );
  }

  async listTripPositions(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<TripMemberPositionView[]>> {
    const res = await this.callProtected<UpstreamTripMemberPosition[]>(
      auth,
      (opts) =>
        this.http.get<UpstreamTripMemberPosition[]>(
          `/location/trips/${encodeURIComponent(tripID)}/members`,
          opts,
        ),
    );
    return mapData(res, (rows) => rows.map(toTripMemberPositionView));
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

const buildSavedListQuery = (q: ListSavedPlacesQueryInput): string => {
  const qs = new URLSearchParams();
  if (q.trip_id) qs.set("trip_id", q.trip_id);
  if (q.category) qs.set("category", q.category);
  return qs.toString();
};

const buildSearchQuery = (q: SearchPlacesQueryInput): string => {
  const qs = new URLSearchParams();
  qs.set("q", q.q);
  if (q.lat !== undefined) qs.set("lat", String(q.lat));
  if (q.lng !== undefined) qs.set("lng", String(q.lng));
  if (q.radius !== undefined) qs.set("radius", String(q.radius));
  if (q.limit !== undefined) qs.set("limit", String(q.limit));
  if (q.lang) qs.set("lang", q.lang);
  return qs.toString();
};

const buildNearbyQuery = (q: NearbyPlacesQueryInput): string => {
  const qs = new URLSearchParams();
  qs.set("lat", String(q.lat));
  qs.set("lng", String(q.lng));
  if (q.radius !== undefined) qs.set("radius", String(q.radius));
  if (q.type) qs.set("type", q.type);
  if (q.limit !== undefined) qs.set("limit", String(q.limit));
  if (q.lang) qs.set("lang", q.lang);
  return qs.toString();
};

const buildWeatherQuery = (q: WeatherQueryInput): string => {
  const qs = new URLSearchParams();
  qs.set("lat", String(q.lat));
  qs.set("lng", String(q.lng));
  if (q.lang) qs.set("lang", q.lang);
  if (q.units) qs.set("units", q.units);
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

export const locationClient = new LocationClient();
