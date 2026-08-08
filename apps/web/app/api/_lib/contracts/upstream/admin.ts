import "server-only";

export type UpstreamAdminSubjectType = "post" | "comment" | "user";
export type UpstreamAdminReportStatus = "pending" | "resolved" | "dismissed";
export type UpstreamAdminUserStatus = "active" | "deactivated" | "deleted";
export type UpstreamAdminPostStatus = "draft" | "published" | "archived";

export type UpstreamAdminUserSummary = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamAdminSubjectView = {
  kind: UpstreamAdminSubjectType;
  id: string;
  title?: string;
  excerpt?: string;
  author_id?: string;
  deleted?: boolean;
};

export type UpstreamElevatePayload = {
  password: string;
};

export type UpstreamElevationResponse = {
  elevated: boolean;
  expires_at: string;
};

export type UpstreamCreateReportPayload = {
  subject_type: UpstreamAdminSubjectType;
  subject_id: string;
  reason: string;
  description?: string;
};

export type UpstreamResolveReportPayload = {
  resolution?: string;
};

export type UpstreamAdminReportResponse = {
  id: string;
  reporter: UpstreamAdminUserSummary;
  subject_type: UpstreamAdminSubjectType;
  subject_id: string;
  subject?: UpstreamAdminSubjectView;
  reason: string;
  description?: string;
  status: UpstreamAdminReportStatus;
  resolution?: string;
  resolved_by?: UpstreamAdminUserSummary;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
};

export type UpstreamAdminListReportsResponse = {
  reports: UpstreamAdminReportResponse[];
  next_cursor?: string;
};

export type UpstreamAdminUserResponse = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  status: UpstreamAdminUserStatus;
  is_blocked: boolean;
  is_verified: boolean;
  roles: string[];
  post_count: number;
  reported_count: number;
  created_at: string;
  deactivated_at?: string;
};

export type UpstreamAdminListUsersResponse = {
  users: UpstreamAdminUserResponse[];
  next_cursor?: string;
};

export type UpstreamAdminPostResponse = {
  id: string;
  author: UpstreamAdminUserSummary;
  title: string;
  content: string;
  status: UpstreamAdminPostStatus;
  like_count: number;
  comment_count: number;
  report_count: number;
  is_deleted: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
};

export type UpstreamAdminListPostsResponse = {
  posts: UpstreamAdminPostResponse[];
  next_cursor?: string;
};

export type UpstreamAdminListUsersQuery = {
  q?: string;
  status?: UpstreamAdminUserStatus;
  is_blocked?: boolean;
  cursor?: string;
  limit?: number;
};

export type UpstreamAdminListPostsQuery = {
  q?: string;
  status?: UpstreamAdminPostStatus;
  author_id?: string;
  include_deleted?: boolean;
  cursor?: string;
  limit?: number;
};

export type UpstreamAdminListReportsQuery = {
  status?: UpstreamAdminReportStatus;
  subject_type?: UpstreamAdminSubjectType;
  cursor?: string;
  limit?: number;
};
