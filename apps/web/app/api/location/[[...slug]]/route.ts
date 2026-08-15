import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { locationClient } from "../../_lib/clients/location-client";
import type {
  ComputeRouteInput,
  CreateSavedPlaceInput,
  ListSavedPlacesQueryInput,
  NearbyPlacesQueryInput,
  SearchPlacesQueryInput,
  ShareTripPositionInput,
  UpdateSavedPlaceInput,
  WeatherQueryInput,
} from "../../_lib/clients/location-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (req: NextRequest, auth: AuthArg) => Promise<ApiResult<unknown>>;

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

export async function PATCH(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPatch(slug);
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
  // GET /location/saved[?trip_id=]
  if (slug.length === 1 && slug[0] === "saved") {
    return (req, auth) =>
      locationClient.listSavedPlaces(auth, readListSavedQuery(req));
  }
  // GET /location/saved/:id
  if (slug.length === 2 && slug[0] === "saved") {
    return (_req, auth) => locationClient.getSavedPlace(slug[1], auth);
  }
  // GET /location/places/search?q=&lat=&lng=&...
  if (slug.length === 2 && slug[0] === "places" && slug[1] === "search") {
    return (req, auth) => {
      const q = readSearchQuery(req);
      if (!q.q) return Promise.resolve(badRequest("q is required"));
      return locationClient.searchPlaces(q, auth);
    };
  }
  // GET /location/places/nearby?lat=&lng=&...
  if (slug.length === 2 && slug[0] === "places" && slug[1] === "nearby") {
    return (req, auth) => {
      const q = readNearbyQuery(req);
      if (q === null) return Promise.resolve(badRequest("lat and lng are required"));
      return locationClient.nearbyPlaces(q, auth);
    };
  }
  // GET /location/places/:place_id
  if (slug.length === 2 && slug[0] === "places") {
    return (req, auth) => {
      const url = new URL(req.url);
      const lang = url.searchParams.get("lang") ?? undefined;
      return locationClient.placeDetails(slug[1], lang, auth);
    };
  }
  // GET /location/weather?lat=&lng=&...
  if (slug.length === 1 && slug[0] === "weather") {
    return (req, auth) => {
      const q = readWeatherQuery(req);
      if (q === null) return Promise.resolve(badRequest("lat and lng are required"));
      return locationClient.weather(q, auth);
    };
  }
  // GET /location/trips/:trip_id/members
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "members") {
    return (_req, auth) => locationClient.listTripPositions(slug[1], auth);
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /location/saved
  if (slug.length === 1 && slug[0] === "saved") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return locationClient.createSavedPlace(body as CreateSavedPlaceInput, auth);
    };
  }
  // POST /location/routes
  if (slug.length === 1 && slug[0] === "routes") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return locationClient.computeRoute(body as ComputeRouteInput, auth);
    };
  }
  // POST /location/trips/:trip_id/share
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "share") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return locationClient.shareTripPosition(
        slug[1],
        body as ShareTripPositionInput,
        auth,
      );
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /location/saved/:id
  if (slug.length === 2 && slug[0] === "saved") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof Response) return responseToResult(body);
      return locationClient.updateSavedPlace(
        slug[1],
        body as UpdateSavedPlaceInput,
        auth,
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /location/saved/:id
  if (slug.length === 2 && slug[0] === "saved") {
    return (_req, auth) => locationClient.deleteSavedPlace(slug[1], auth);
  }
  // DELETE /location/trips/:trip_id/share
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "share") {
    return (_req, auth) => locationClient.revokeTripPosition(slug[1], auth);
  }
  return null;
};

const readListSavedQuery = (req: NextRequest): ListSavedPlacesQueryInput => {
  const url = new URL(req.url);
  const out: ListSavedPlacesQueryInput = {};
  const trip = url.searchParams.get("trip_id");
  if (trip) out.trip_id = trip;
  return out;
};

const readSearchQuery = (req: NextRequest): SearchPlacesQueryInput => {
  const url = new URL(req.url);
  const out: SearchPlacesQueryInput = { q: url.searchParams.get("q") ?? "" };
  const lat = num(url.searchParams.get("lat"));
  const lng = num(url.searchParams.get("lng"));
  if (lat !== undefined) out.lat = lat;
  if (lng !== undefined) out.lng = lng;
  const radius = int(url.searchParams.get("radius"));
  if (radius !== undefined) out.radius = radius;
  const limit = int(url.searchParams.get("limit"));
  if (limit !== undefined) out.limit = limit;
  const lang = url.searchParams.get("lang");
  if (lang) out.lang = lang;
  return out;
};

const readNearbyQuery = (req: NextRequest): NearbyPlacesQueryInput | null => {
  const url = new URL(req.url);
  const lat = num(url.searchParams.get("lat"));
  const lng = num(url.searchParams.get("lng"));
  if (lat === undefined || lng === undefined) return null;
  const out: NearbyPlacesQueryInput = { lat, lng };
  const radius = int(url.searchParams.get("radius"));
  if (radius !== undefined) out.radius = radius;
  const type = url.searchParams.get("type");
  if (type) out.type = type;
  const limit = int(url.searchParams.get("limit"));
  if (limit !== undefined) out.limit = limit;
  const lang = url.searchParams.get("lang");
  if (lang) out.lang = lang;
  return out;
};

const readWeatherQuery = (req: NextRequest): WeatherQueryInput | null => {
  const url = new URL(req.url);
  const lat = num(url.searchParams.get("lat"));
  const lng = num(url.searchParams.get("lng"));
  if (lat === undefined || lng === undefined) return null;
  const out: WeatherQueryInput = { lat, lng };
  const lang = url.searchParams.get("lang");
  if (lang) out.lang = lang;
  const units = url.searchParams.get("units");
  if (units === "metric" || units === "imperial") out.units = units;
  return out;
};

const num = (s: string | null): number | undefined => {
  if (s === null || s === "") return undefined;
  const v = Number(s);
  return Number.isFinite(v) ? v : undefined;
};

const int = (s: string | null): number | undefined => {
  if (s === null || s === "") return undefined;
  const v = parseInt(s, 10);
  return Number.isFinite(v) ? v : undefined;
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

const badRequest = (message: string): ApiResult<unknown> => ({
  ok: false,
  status: 400,
  code: 400,
  message,
  error: {
    category: "validation",
    status: 400,
    code: 400,
    message,
    retryable: false,
  },
  payload: null,
  raw: "",
});

const notFound = (method: string, slug: string[]): NextResponse =>
  NextResponse.json(
    {
      code: 404,
      message: `unknown location op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
