export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  created_at: string;
};

export type SessionView = {
  user: User;
  expires_in: number;
  token_type?: string;
  requires_totp?: boolean;
  requires_verification?: boolean;
};

type Envelope<T> = { code: number; message: string; data: T | null };

export class ApiError extends Error {
  status: number;
  code?: number;
  constructor(message: string, status: number, code?: number) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const { json, headers, ...rest } = init;
  const res = await fetch(path, {
    credentials: "include",
    ...rest,
    headers: {
      ...(json !== undefined ? { "content-type": "application/json" } : {}),
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  });

  const body = (await res.json().catch(() => null)) as Envelope<T> | null;
  if (!res.ok) {
    throw new ApiError(
      body?.message ?? `Request failed (${res.status})`,
      res.status,
      body?.code,
    );
  }
  return (body?.data as T) ?? (null as T);
}

export const authApi = {
  signIn: (input: { email: string; password: string }, signal?: AbortSignal) =>
    request<SessionView>("/api/auth/login", {
      method: "POST",
      json: input,
      signal,
    }),
  signUp: (
    input: { name: string; email: string; password: string },
    signal?: AbortSignal,
  ) =>
    request<SessionView>("/api/auth/register", {
      method: "POST",
      json: input,
      signal,
    }),
  forgotPassword: (input: { email: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/forgot-password", {
      method: "POST",
      json: input,
      signal,
    }),
  resetPassword: (
    input: { token: string; password: string },
    signal?: AbortSignal,
  ) =>
    request<null>("/api/auth/reset-password", {
      method: "POST",
      json: input,
      signal,
    }),
  verifyEmail: (input: { token: string }, signal?: AbortSignal) =>
    request<SessionView>("/api/auth/verify-email", {
      method: "POST",
      json: input,
      signal,
    }),
  resendVerification: (input: { email: string }, signal?: AbortSignal) =>
    request<null>("/api/auth/resend-verification", {
      method: "POST",
      json: input,
      signal,
    }),
  me: (signal?: AbortSignal) => request<User>("/api/auth/me", { signal }),
};
