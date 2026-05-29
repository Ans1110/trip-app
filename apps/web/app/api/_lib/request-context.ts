import { NextRequest } from "next/server";
import "server-only";

export type RequestContext = {
  requestId: string;
  clientIp?: string;
  userAgent?: string;
  deviceName?: string;
  deviceType?: string;
};

export const extractRequestContext = (req: NextRequest): RequestContext => {
  const xff = req.headers.get("x-forwarded-for");
  const clientIp =
    xff?.split(",")[0]?.trim() ?? req.headers.get("x-real-ip") ?? undefined;
  return {
    requestId: req.headers.get("x-request-id") ?? crypto.randomUUID(),
    clientIp,
    userAgent: req.headers.get("user-agent") ?? undefined,
    deviceName: req.headers.get("x-device-name") ?? undefined,
    deviceType: req.headers.get("x-device-type") ?? undefined,
  };
};
