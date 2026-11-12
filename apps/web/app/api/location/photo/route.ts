import "server-only";
import { NextRequest, NextResponse } from "next/server";

import { locationClient } from "../../_lib/clients/location-client";
import { extractRequestContext } from "../../_lib/request-context";

const NAME_RE = /^places\/[A-Za-z0-9_-]+\/photos\/[A-Za-z0-9_-]+$/;

// Redirects to Google's short-lived image URL so <img src> can point here
// while the Places API key stays on the Go API.
export async function GET(req: NextRequest) {
  const name = req.nextUrl.searchParams.get("name") ?? "";
  const width = Number(req.nextUrl.searchParams.get("w") ?? 480);
  if (!NAME_RE.test(name)) {
    return NextResponse.json({ code: 400, message: "invalid photo name" }, { status: 400 });
  }

  const result = await locationClient.placePhoto(
    name,
    Number.isFinite(width) ? Math.min(Math.max(Math.round(width), 64), 1200) : 480,
    { ctx: extractRequestContext(req), signal: req.signal },
  );
  if (!result.ok || !result.data?.url) {
    return new NextResponse(null, { status: result.ok ? 404 : result.status || 502 });
  }
  const res = NextResponse.redirect(result.data.url, 302);
  res.headers.set("Cache-Control", "private, max-age=1800");
  return res;
}
