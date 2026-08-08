import "server-only";
import {
  UpstreamAdminListPostsResponse,
  UpstreamAdminListReportsResponse,
  UpstreamAdminListUsersResponse,
  UpstreamAdminPostResponse,
  UpstreamAdminPostStatus,
  UpstreamAdminReportResponse,
  UpstreamAdminReportStatus,
  UpstreamAdminSubjectType,
  UpstreamAdminSubjectView,
  UpstreamAdminUserResponse,
  UpstreamAdminUserStatus,
  UpstreamAdminUserSummary,
  UpstreamElevationResponse,
} from "../upstream";

export type AdminSubjectTypeView = UpstreamAdminSubjectType;
export type AdminReportStatusView = UpstreamAdminReportStatus;
export type AdminUserStatusView = UpstreamAdminUserStatus;
export type AdminPostStatusView = UpstreamAdminPostStatus;

export type AdminUserSummaryView = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type ElevationView = {
  elevated: boolean;
  expires_at: string;
};

export type AdminSubjectView = {
  kind: AdminSubjectTypeView;
  id: string;
  title?: string;
  excerpt?: string;
  author_id?: string;
  deleted?: boolean;
};

export type AdminReportView = {
  id: string;
  reporter: AdminUserSummaryView;
  subject_type: AdminSubjectTypeView;
  subject_id: string;
  subject?: AdminSubjectView;
  reason: string;
  description?: string;
  status: AdminReportStatusView;
  resolution?: string;
  resolved_by?: AdminUserSummaryView;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminListReportsView = {
  reports: AdminReportView[];
  next_cursor?: string;
};

export type AdminUserView = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  status: AdminUserStatusView;
  is_blocked: boolean;
  is_verified: boolean;
  roles: string[];
  post_count: number;
  reported_count: number;
  created_at: string;
  deactivated_at?: string;
};

export type AdminListUsersView = {
  users: AdminUserView[];
  next_cursor?: string;
};

export type AdminPostView = {
  id: string;
  author: AdminUserSummaryView;
  title: string;
  content: string;
  status: AdminPostStatusView;
  like_count: number;
  comment_count: number;
  report_count: number;
  is_deleted: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
};

export type AdminListPostsView = {
  posts: AdminPostView[];
  next_cursor?: string;
};

export const toAdminUserSummaryView = (
  u: UpstreamAdminUserSummary,
): AdminUserSummaryView => ({
  id: u.id,
  email: u.email,
  name: u.name,
  avatar_url: u.avatar_url,
});

export const toElevationView = (r: UpstreamElevationResponse): ElevationView => ({
  elevated: r.elevated,
  expires_at: r.expires_at,
});

const toSubjectView = (
  s: UpstreamAdminSubjectView,
): AdminSubjectView => ({
  kind: s.kind,
  id: s.id,
  title: s.title,
  excerpt: s.excerpt,
  author_id: s.author_id,
  deleted: s.deleted,
});

export const toAdminReportView = (
  r: UpstreamAdminReportResponse,
): AdminReportView => ({
  id: r.id,
  reporter: toAdminUserSummaryView(r.reporter),
  subject_type: r.subject_type,
  subject_id: r.subject_id,
  subject: r.subject ? toSubjectView(r.subject) : undefined,
  reason: r.reason,
  description: r.description,
  status: r.status,
  resolution: r.resolution,
  resolved_by: r.resolved_by ? toAdminUserSummaryView(r.resolved_by) : undefined,
  resolved_at: r.resolved_at,
  created_at: r.created_at,
  updated_at: r.updated_at,
});

export const toAdminListReportsView = (
  r: UpstreamAdminListReportsResponse,
): AdminListReportsView => ({
  reports: (r.reports ?? []).map(toAdminReportView),
  next_cursor: r.next_cursor,
});

export const toAdminUserView = (u: UpstreamAdminUserResponse): AdminUserView => ({
  id: u.id,
  email: u.email,
  name: u.name,
  avatar_url: u.avatar_url,
  status: u.status,
  is_blocked: u.is_blocked,
  is_verified: u.is_verified,
  roles: u.roles ?? [],
  post_count: u.post_count ?? 0,
  reported_count: u.reported_count ?? 0,
  created_at: u.created_at,
  deactivated_at: u.deactivated_at,
});

export const toAdminListUsersView = (
  r: UpstreamAdminListUsersResponse,
): AdminListUsersView => ({
  users: (r.users ?? []).map(toAdminUserView),
  next_cursor: r.next_cursor,
});

export const toAdminPostView = (p: UpstreamAdminPostResponse): AdminPostView => ({
  id: p.id,
  author: toAdminUserSummaryView(p.author),
  title: p.title,
  content: p.content,
  status: p.status,
  like_count: p.like_count ?? 0,
  comment_count: p.comment_count ?? 0,
  report_count: p.report_count ?? 0,
  is_deleted: p.is_deleted,
  published_at: p.published_at,
  created_at: p.created_at,
  updated_at: p.updated_at,
});

export const toAdminListPostsView = (
  r: UpstreamAdminListPostsResponse,
): AdminListPostsView => ({
  posts: (r.posts ?? []).map(toAdminPostView),
  next_cursor: r.next_cursor,
});
