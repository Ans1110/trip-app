import { ApiError, type Envelope } from "./utils";

type SignaturePayload = {
  cloud_name: string;
  api_key: string;
  timestamp: number;
  folder: string;
  signature: string;
  upload_url: string;
  timeout_ms: number;
};

export type UploadedImage = {
  secure_url: string;
  public_id: string;
  width: number;
  height: number;
  format: string;
  bytes: number;
};

const fetchSignature = async (
  signal?: AbortSignal,
): Promise<SignaturePayload> => {
  const res = await fetch("/api/uploads/sign", {
    method: "POST",
    credentials: "include",
    signal,
  });
  const body = (await res.json().catch(() => null)) as
    | Envelope<SignaturePayload>
    | null;
  if (!res.ok || !body?.data) {
    throw new ApiError(
      body?.message ?? `Sign request failed (${res.status})`,
      res.status,
      body?.code,
    );
  }
  return body.data;
};

const linkSignals = (
  user: AbortSignal | undefined,
  timeoutMs: number,
): AbortSignal => {
  const timeout = AbortSignal.timeout(timeoutMs);
  if (!user) return timeout;
  const anyFn = (
    AbortSignal as unknown as { any?: (signals: AbortSignal[]) => AbortSignal }
  ).any;
  return typeof anyFn === "function" ? anyFn([user, timeout]) : timeout;
};

export async function uploadImage(
  file: File,
  options?: { signal?: AbortSignal },
): Promise<UploadedImage> {
  const sig = await fetchSignature(options?.signal);

  const form = new FormData();
  form.set("file", file);
  form.set("api_key", sig.api_key);
  form.set("timestamp", String(sig.timestamp));
  form.set("folder", sig.folder);
  form.set("signature", sig.signature);

  const signal = linkSignals(options?.signal, sig.timeout_ms);
  const res = await fetch(sig.upload_url, {
    method: "POST",
    body: form,
    signal,
  });

  const payload = (await res.json().catch(() => null)) as
    | (UploadedImage & { error?: { message?: string } })
    | null;
  if (!res.ok || !payload?.secure_url) {
    throw new ApiError(
      payload?.error?.message ?? `Upload failed (${res.status})`,
      res.status,
    );
  }
  return {
    secure_url: payload.secure_url,
    public_id: payload.public_id,
    width: payload.width,
    height: payload.height,
    format: payload.format,
    bytes: payload.bytes,
  };
}
