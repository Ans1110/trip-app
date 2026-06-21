import "server-only";
import { NextResponse } from "next/server";

import { buildUploadSignature } from "../../_lib/cloudinary";
import { rootLogger } from "../../_lib/logger";
import { readSessionTokens } from "../../_lib/session-store";

const log = rootLogger.child({ module: "uploads-sign" });

export async function POST() {
  const tokens = await readSessionTokens();
  if (!tokens.accessToken) {
    return NextResponse.json(
      { code: 401, message: "not authenticated" },
      { status: 401 },
    );
  }

  try {
    const sig = buildUploadSignature();
    return NextResponse.json({ code: 200, message: "ok", data: sig });
  } catch (err) {
    log.error("uploads.sign_failed", {
      reason: err instanceof Error ? err.message : "unknown",
    });
    return NextResponse.json(
      { code: 500, message: "upload signing unavailable" },
      { status: 500 },
    );
  }
}
