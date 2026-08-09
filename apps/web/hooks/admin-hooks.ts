"use client";

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  adminApi,
  type AdminReport,
  type AdminUser,
  type CreateReportInput,
  type Elevation,
  type ElevateInput,
  type ListAdminPosts,
  type ListAdminPostsQuery,
  type ListAdminReports,
  type ListAdminReportsQuery,
  type ListAdminUsers,
  type ListAdminUsersQuery,
  type ResolveReportInput,
} from "@/lib/apis/admin-api";
import { ApiError } from "@/lib/utils";
import { authKeys } from "@/hooks/auth-hooks";

export const adminKeys = {
  all: ["admin"] as const,
  users: () => [...adminKeys.all, "users"] as const,
  userInfinite: (query?: Omit<ListAdminUsersQuery, "cursor">) =>
    [
      ...adminKeys.users(),
      "infinite",
      query?.q ?? null,
      query?.status ?? null,
      query?.is_blocked ?? null,
      query?.limit ?? null,
    ] as const,
  user: (id: string) => [...adminKeys.users(), "detail", id] as const,
  posts: () => [...adminKeys.all, "posts"] as const,
  postInfinite: (query?: Omit<ListAdminPostsQuery, "cursor">) =>
    [
      ...adminKeys.posts(),
      "infinite",
      query?.q ?? null,
      query?.status ?? null,
      query?.author_id ?? null,
      query?.include_deleted ?? null,
      query?.limit ?? null,
    ] as const,
  reports: () => [...adminKeys.all, "reports"] as const,
  reportInfinite: (query?: Omit<ListAdminReportsQuery, "cursor">) =>
    [
      ...adminKeys.reports(),
      "infinite",
      query?.status ?? null,
      query?.subject_type ?? null,
      query?.limit ?? null,
    ] as const,
};

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

export function useElevate(options?: MutOpts<Elevation, ElevateInput>) {
  const qc = useQueryClient();
  return useMutation<Elevation, ApiError, ElevateInput>({
    mutationFn: (input) => adminApi.elevate(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: authKeys.me });
      qc.invalidateQueries({ queryKey: adminKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAdminLogout(options?: MutOpts<null, void>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, void>({
    mutationFn: () => adminApi.adminLogout(),
    ...options,
    onSuccess: (...args) => {
      qc.removeQueries({ queryKey: adminKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useSubmitReport(
  options?: MutOpts<AdminReport, CreateReportInput>,
) {
  return useMutation<AdminReport, ApiError, CreateReportInput>({
    mutationFn: (input) => adminApi.submitReport(input),
    ...options,
  });
}

export function useAdminUsers(
  query?: Omit<ListAdminUsersQuery, "cursor">,
  options?: QueryOpts<ListAdminUsers>,
) {
  const key = adminKeys.userInfinite(query);
  const infinite = useInfiniteQuery<ListAdminUsers, ApiError>({
    queryKey: key,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      adminApi.listUsers(
        { ...(query ?? {}), cursor: pageParam as string | undefined },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    ...(options as Record<string, unknown>),
  });
  return infinite;
}

export function useAdminUser(id: string, options?: QueryOpts<AdminUser>) {
  return useQuery<AdminUser, ApiError>({
    queryKey: adminKeys.user(id),
    queryFn: ({ signal }) => adminApi.getUser(id, signal),
    enabled: id.length > 0,
    ...options,
  });
}

export function useBlockUser(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.blockUser(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: adminKeys.users() });
      qc.invalidateQueries({ queryKey: adminKeys.user(id) });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUnblockUser(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.unblockUser(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: adminKeys.users() });
      qc.invalidateQueries({ queryKey: adminKeys.user(id) });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useDeactivateUser(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.deactivateUser(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: adminKeys.users() });
      qc.invalidateQueries({ queryKey: adminKeys.user(id) });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useReactivateUser(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.reactivateUser(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: adminKeys.users() });
      qc.invalidateQueries({ queryKey: adminKeys.user(id) });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAdminPosts(
  query?: Omit<ListAdminPostsQuery, "cursor">,
  options?: QueryOpts<ListAdminPosts>,
) {
  return useInfiniteQuery<ListAdminPosts, ApiError>({
    queryKey: adminKeys.postInfinite(query),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      adminApi.listPosts(
        { ...(query ?? {}), cursor: pageParam as string | undefined },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    ...(options as Record<string, unknown>),
  });
}

export function useAdminDeletePost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.deletePost(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: adminKeys.posts() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAdminRestorePost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.restorePost(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: adminKeys.posts() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAdminDeleteComment(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => adminApi.deleteComment(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: adminKeys.reports() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useAdminReports(
  query?: Omit<ListAdminReportsQuery, "cursor">,
  options?: QueryOpts<ListAdminReports>,
) {
  return useInfiniteQuery<ListAdminReports, ApiError>({
    queryKey: adminKeys.reportInfinite(query),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      adminApi.listReports(
        { ...(query ?? {}), cursor: pageParam as string | undefined },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    ...(options as Record<string, unknown>),
  });
}

type ResolveVars = { id: string; input: ResolveReportInput };

export function useResolveReport(options?: MutOpts<AdminReport, ResolveVars>) {
  const qc = useQueryClient();
  return useMutation<AdminReport, ApiError, ResolveVars>({
    mutationFn: ({ id, input }) => adminApi.resolveReport(id, input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: adminKeys.reports() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useDismissReport(options?: MutOpts<AdminReport, ResolveVars>) {
  const qc = useQueryClient();
  return useMutation<AdminReport, ApiError, ResolveVars>({
    mutationFn: ({ id, input }) => adminApi.dismissReport(id, input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: adminKeys.reports() });
      return options?.onSuccess?.(...args);
    },
  });
}

export { errorMessage } from "@/lib/utils";
