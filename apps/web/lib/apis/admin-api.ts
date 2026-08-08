import { request } from "../utils";

export type AdminSubjectType = "post" | "comment" | "user";
export type AdminReportStatus = "pending" | "resolved" | "dismissed";
export type AdminUserStatus = "active" | "deactivated" | "deleted";
export type AdminPostStatus = "draft" | "published" | "archived";

export type AdminUserSummary = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type Elevation = {
  elevated: boolean;
  expires_at: string;
};

export type AdminSubject = {
  kind: AdminSubjectType;
  id: string;
  title?: string;
  excerpt?: string;
  author_id?: string;
  deleted?: boolean;
};

export type AdminReport = {
  id: string;
  reporter: AdminUserSummary;
  subject_type: AdminSubjectType;
  subject_id: string;
  subject?: AdminSubject;
  reason: string;
  description?: string;
  status: AdminReportStatus;
  resolution?: string;
  resolved_by?: AdminUserSummary;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
};

export type ListAdminReports = {
  reports: AdminReport[];
  next_cursor?: string;
};

export type AdminUser = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  status: AdminUserStatus;
  is_blocked: boolean;
  is_verified: boolean;
  roles: string[];
  post_count: number;
  reported_count: number;
  created_at: string;
  deactivated_at?: string;
};

export type ListAdminUsers = {
  users: AdminUser[];
  next_cursor?: string;
};

export type AdminPost = {
  id: string;
  author: AdminUserSummary;
  title: string;
  content: string;
  status: AdminPostStatus;
  like_count: number;
  comment_count: number;
  report_count: number;
  is_deleted: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
};

export type ListAdminPosts = {
  posts: AdminPost[];
  next_cursor?: string;
};

export type ElevateInput = { password: string };
export type CreateReportInput = {
  subject_type: AdminSubjectType;
  subject_id: string;
  reason: string;
  description?: string;
};
export type ResolveReportInput = { resolution?: string };

export type ListAdminUsersQuery = {
  q?: string;
  status?: AdminUserStatus;
  is_blocked?: boolean;
  cursor?: string;
  limit?: number;
};

export type ListAdminPostsQuery = {
  q?: string;
  status?: AdminPostStatus;
  author_id?: string;
  include_deleted?: boolean;
  cursor?: string;
  limit?: number;
};

export type ListAdminReportsQuery = {
  status?: AdminReportStatus;
  subject_type?: AdminSubjectType;
  cursor?: string;
  limit?: number;
};

const usersQS = (q?: ListAdminUsersQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.q) qs.set("q", q.q);
  if (q.status) qs.set("status", q.status);
  if (typeof q.is_blocked === "boolean") qs.set("is_blocked", String(q.is_blocked));
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

const postsQS = (q?: ListAdminPostsQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.q) qs.set("q", q.q);
  if (q.status) qs.set("status", q.status);
  if (q.author_id) qs.set("author_id", q.author_id);
  if (typeof q.include_deleted === "boolean") {
    qs.set("include_deleted", String(q.include_deleted));
  }
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

const reportsQS = (q?: ListAdminReportsQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.status) qs.set("status", q.status);
  if (q.subject_type) qs.set("subject_type", q.subject_type);
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

export const adminApi = {
  elevate: (input: ElevateInput, signal?: AbortSignal) =>
    request<Elevation>("/api/admin/auth/elevate", {
      method: "POST",
      json: input,
      signal,
    }),

  adminLogout: (signal?: AbortSignal) =>
    request<null>("/api/admin/auth/logout", { method: "POST", signal }),

  submitReport: (input: CreateReportInput, signal?: AbortSignal) =>
    request<AdminReport>("/api/reports", {
      method: "POST",
      json: input,
      signal,
    }),

  listUsers: (query?: ListAdminUsersQuery, signal?: AbortSignal) =>
    request<ListAdminUsers>(`/api/admin/users${usersQS(query)}`, { signal }),

  getUser: (id: string, signal?: AbortSignal) =>
    request<AdminUser>(`/api/admin/users/${encodeURIComponent(id)}`, { signal }),

  blockUser: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/users/${encodeURIComponent(id)}/block`, {
      method: "POST",
      signal,
    }),

  unblockUser: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/users/${encodeURIComponent(id)}/unblock`, {
      method: "POST",
      signal,
    }),

  deactivateUser: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/users/${encodeURIComponent(id)}/deactivate`, {
      method: "POST",
      signal,
    }),

  reactivateUser: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/users/${encodeURIComponent(id)}/reactivate`, {
      method: "POST",
      signal,
    }),

  listPosts: (query?: ListAdminPostsQuery, signal?: AbortSignal) =>
    request<ListAdminPosts>(`/api/admin/posts${postsQS(query)}`, { signal }),

  deletePost: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/posts/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  restorePost: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/posts/${encodeURIComponent(id)}/restore`, {
      method: "POST",
      signal,
    }),

  deleteComment: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/admin/comments/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  listReports: (query?: ListAdminReportsQuery, signal?: AbortSignal) =>
    request<ListAdminReports>(`/api/admin/reports${reportsQS(query)}`, {
      signal,
    }),

  resolveReport: (
    id: string,
    input: ResolveReportInput,
    signal?: AbortSignal,
  ) =>
    request<AdminReport>(
      `/api/admin/reports/${encodeURIComponent(id)}/resolve`,
      { method: "POST", json: input, signal },
    ),

  dismissReport: (
    id: string,
    input: ResolveReportInput,
    signal?: AbortSignal,
  ) =>
    request<AdminReport>(
      `/api/admin/reports/${encodeURIComponent(id)}/dismiss`,
      { method: "POST", json: input, signal },
    ),
};
