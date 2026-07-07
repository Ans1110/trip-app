"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { authApi, type SessionView, type User } from "@/lib/apis/auth-api";
import { ApiError } from "@/lib/utils";
import type {
  ForgotPasswordInput,
  SignInInput,
  SignUpInput,
} from "@/lib/schemas/auth-schemas";

export const authKeys = {
  me: ["auth", "me"] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useMe(options?: QueryOpts<User>) {
  return useQuery<User, ApiError>({
    queryKey: authKeys.me,
    queryFn: ({ signal }) => authApi.me(signal),
    retry: false,
    ...options,
  });
}

export function useSignIn(options?: MutOpts<SessionView, SignInInput>) {
  return useMutation<SessionView, ApiError, SignInInput>({
    mutationFn: (input) => authApi.signIn(input),
    ...options,
  });
}

export function useSignUp(options?: MutOpts<SessionView, SignUpInput>) {
  return useMutation<SessionView, ApiError, SignUpInput>({
    mutationFn: (input) => authApi.signUp(input),
    ...options,
  });
}

export function useForgotPassword(
  options?: MutOpts<null, ForgotPasswordInput>,
) {
  return useMutation<null, ApiError, ForgotPasswordInput>({
    mutationFn: (input) => authApi.forgotPassword(input),
    ...options,
  });
}

type ResetPasswordVars = { token: string; password: string };

export function useResetPassword(options?: MutOpts<null, ResetPasswordVars>) {
  return useMutation<null, ApiError, ResetPasswordVars>({
    mutationFn: (input) => authApi.resetPassword(input),
    ...options,
  });
}

export function useVerifyEmail(options?: MutOpts<SessionView, string>) {
  const qc = useQueryClient();
  return useMutation<SessionView, ApiError, string>({
    mutationFn: (token) => authApi.verifyEmail({ token }),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: authKeys.me });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useResendVerification(options?: MutOpts<null, string>) {
  return useMutation<null, ApiError, string>({
    mutationFn: (email) => authApi.resendVerification({ email }),
    ...options,
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
