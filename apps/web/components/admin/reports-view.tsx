"use client";

import { useState } from "react";
import { CheckCircle2, Loader2, Trash2, XCircle } from "lucide-react";

import {
  errorMessage,
  useAdminDeleteComment,
  useAdminDeletePost,
  useAdminReports,
  useDismissReport,
  useResolveReport,
} from "@/hooks/admin-hooks";
import type {
  AdminReport,
  AdminReportStatus,
  AdminSubjectType,
  ListAdminReportsQuery,
} from "@/lib/apis/admin-api";

const STATUS_OPTIONS: { label: string; value: AdminReportStatus | "" }[] = [
  { label: "All", value: "" },
  { label: "Pending", value: "pending" },
  { label: "Resolved", value: "resolved" },
  { label: "Dismissed", value: "dismissed" },
];

const SUBJECT_OPTIONS: { label: string; value: AdminSubjectType | "" }[] = [
  { label: "All subjects", value: "" },
  { label: "Posts", value: "post" },
  { label: "Comments", value: "comment" },
  { label: "Users", value: "user" },
];

export function ReportsView() {
  const [query, setQuery] = useState<Omit<ListAdminReportsQuery, "cursor">>({
    limit: 20,
    status: "pending",
  });
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useAdminReports(query);

  if (isLoading) return <LoadingState label="Loading reports" />;
  if (error) return <ErrorState message={errorMessage(error)} />;

  const reports = data?.pages.flatMap((p) => p.reports) ?? [];

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <select
          value={query.status ?? ""}
          onChange={(e) =>
            setQuery((prev) => ({
              ...prev,
              status:
                e.target.value === ""
                  ? undefined
                  : (e.target.value as AdminReportStatus),
            }))
          }
          className="px-3 py-2 text-sm rounded-lg border bg-transparent"
          style={{ borderColor: "var(--border)", color: "var(--foreground)" }}
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value} className="bg-card">
              {o.label}
            </option>
          ))}
        </select>
        <select
          value={query.subject_type ?? ""}
          onChange={(e) =>
            setQuery((prev) => ({
              ...prev,
              subject_type:
                e.target.value === ""
                  ? undefined
                  : (e.target.value as AdminSubjectType),
            }))
          }
          className="px-3 py-2 text-sm rounded-lg border bg-transparent"
          style={{ borderColor: "var(--border)", color: "var(--foreground)" }}
        >
          {SUBJECT_OPTIONS.map((o) => (
            <option key={o.value} value={o.value} className="bg-card">
              {o.label}
            </option>
          ))}
        </select>
      </div>

      {reports.length === 0 ? (
        <EmptyState message="No reports match this filter." />
      ) : (
        <ul className="flex flex-col gap-3">
          {reports.map((r) => (
            <ReportRow key={r.id} report={r} />
          ))}
        </ul>
      )}

      {hasNextPage && (
        <div className="mt-6 flex justify-center">
          <button
            type="button"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
            className="px-4 py-2 text-sm rounded-full border hover:bg-muted disabled:opacity-60"
            style={{ borderColor: "var(--border)", color: "var(--foreground)" }}
          >
            {isFetchingNextPage ? "Loading…" : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}

function ReportRow({ report }: { report: AdminReport }) {
  const [expanded, setExpanded] = useState(false);
  const [note, setNote] = useState("");
  const resolve = useResolveReport();
  const dismiss = useDismissReport();
  const deletePost = useAdminDeletePost();
  const deleteComment = useAdminDeleteComment();

  const isBusy =
    resolve.isPending ||
    dismiss.isPending ||
    deletePost.isPending ||
    deleteComment.isPending;

  const pending = report.status === "pending";
  const subject = report.subject;

  return (
    <li
      className="flex flex-col gap-3 p-4 rounded-xl border"
      style={{ backgroundColor: "var(--card)", borderColor: "var(--border)" }}
    >
      <div className="flex flex-wrap items-center gap-2">
        <StatusPill status={report.status} />
        <SubjectPill type={report.subject_type} />
        <span className="text-xs" style={{ color: "var(--muted-foreground)" }}>
          reason: <span style={{ color: "var(--foreground)" }}>{report.reason}</span>
        </span>
        <span className="text-[11px] ml-auto" style={{ color: "var(--subtle-foreground)" }}>
          {new Date(report.created_at).toLocaleString()}
        </span>
      </div>

      {report.description && (
        <p
          className="text-sm whitespace-pre-wrap"
          style={{ color: "var(--muted-foreground)" }}
        >
          {report.description}
        </p>
      )}

      <div
        className="text-xs rounded-lg p-3 border"
        style={{
          borderColor: "var(--border)",
          backgroundColor: "var(--surface-sunken)",
          color: "var(--muted-foreground)",
        }}
      >
        <p>
          <span style={{ color: "var(--foreground)" }}>Reporter:</span>{" "}
          {report.reporter.name || report.reporter.email}
        </p>
        <p className="mt-1">
          <span style={{ color: "var(--foreground)" }}>Subject:</span>{" "}
          {subject?.title || subject?.excerpt || report.subject_id}
          {subject?.deleted && (
            <span className="ml-2" style={{ color: "var(--destructive)" }}>
              (deleted)
            </span>
          )}
        </p>
        {report.status !== "pending" && report.resolved_by && (
          <p className="mt-1">
            <span style={{ color: "var(--foreground)" }}>
              {report.status === "resolved" ? "Resolved" : "Dismissed"} by:
            </span>{" "}
            {report.resolved_by.name || report.resolved_by.email}
            {report.resolved_at &&
              ` · ${new Date(report.resolved_at).toLocaleString()}`}
          </p>
        )}
        {report.resolution && (
          <p className="mt-1">
            <span style={{ color: "var(--foreground)" }}>Note:</span> {report.resolution}
          </p>
        )}
      </div>

      {pending && (
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap gap-2">
            <ActionButton
              label={expanded ? "Cancel" : "Add note"}
              onClick={() => setExpanded((v) => !v)}
              disabled={isBusy}
            />
            {report.subject_type === "post" && subject && !subject.deleted && (
              <ActionButton
                label="Delete post"
                danger
                icon={<Trash2 className="size-3.5" />}
                onClick={() => {
                  if (confirm("Delete this post?")) {
                    deletePost.mutate(report.subject_id);
                  }
                }}
                pending={deletePost.isPending}
                disabled={isBusy}
              />
            )}
            {report.subject_type === "comment" &&
              subject &&
              !subject.deleted && (
                <ActionButton
                  label="Delete comment"
                  danger
                  icon={<Trash2 className="size-3.5" />}
                  onClick={() => {
                    if (confirm("Delete this comment?")) {
                      deleteComment.mutate(report.subject_id);
                    }
                  }}
                  pending={deleteComment.isPending}
                  disabled={isBusy}
                />
              )}
            <ActionButton
              label="Dismiss"
              icon={<XCircle className="size-3.5" />}
              onClick={() =>
                dismiss.mutate({
                  id: report.id,
                  input: { resolution: note.trim() || undefined },
                })
              }
              pending={dismiss.isPending}
              disabled={isBusy}
            />
            <ActionButton
              label="Resolve"
              icon={<CheckCircle2 className="size-3.5" />}
              onClick={() =>
                resolve.mutate({
                  id: report.id,
                  input: { resolution: note.trim() || undefined },
                })
              }
              pending={resolve.isPending}
              disabled={isBusy}
            />
          </div>
          {expanded && (
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Optional resolution note"
              rows={2}
              maxLength={1000}
              className="w-full px-3 py-2 text-sm rounded-lg border bg-transparent outline-none focus:ring-2 focus:ring-ring/40 resize-none"
              style={{ borderColor: "var(--border)", color: "var(--foreground)" }}
            />
          )}
          {(resolve.error || dismiss.error) && (
            <p className="text-xs" style={{ color: "var(--destructive)" }}>
              {errorMessage(resolve.error) ?? errorMessage(dismiss.error)}
            </p>
          )}
        </div>
      )}
    </li>
  );
}

function StatusPill({ status }: { status: AdminReportStatus }) {
  if (status === "pending") return <Pill label="pending" tone="warn" />;
  if (status === "resolved") return <Pill label="resolved" tone="ok" />;
  return <Pill label="dismissed" tone="muted" />;
}

function SubjectPill({ type }: { type: AdminSubjectType }) {
  return <Pill label={type} tone="accent" />;
}

function Pill({
  label,
  tone,
}: {
  label: string;
  tone: "ok" | "warn" | "danger" | "muted" | "accent";
}) {
  const styles: Record<typeof tone, { bg: string; fg: string }> = {
    ok: { bg: "color-mix(in oklch, var(--mint) 24%, transparent)", fg: "var(--mint-ink)" },
    warn: { bg: "rgba(250,204,21,0.12)", fg: "var(--warning-ink)" },
    danger: { bg: "rgba(220,38,38,0.12)", fg: "var(--destructive)" },
    muted: { bg: "color-mix(in oklch, var(--muted-foreground) 12%, transparent)", fg: "var(--muted-foreground)" },
    accent: { bg: "color-mix(in oklch, var(--mint) 36%, transparent)", fg: "var(--season-button)" },
  };
  const s = styles[tone];
  return (
    <span
      className="text-[10px] uppercase tracking-wider px-1.5 py-0.5 rounded"
      style={{ backgroundColor: s.bg, color: s.fg }}
    >
      {label}
    </span>
  );
}

function ActionButton({
  label,
  onClick,
  pending,
  disabled,
  danger,
  icon,
}: {
  label: string;
  onClick: () => void;
  pending?: boolean;
  disabled?: boolean;
  danger?: boolean;
  icon?: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs rounded-full border hover:bg-muted disabled:opacity-60"
      style={{
        borderColor: danger ? "rgba(220,38,38,0.35)" : "var(--border)",
        color: danger ? "var(--destructive)" : "var(--foreground)",
      }}
    >
      {pending ? <Loader2 className="size-3.5 animate-spin" /> : icon}
      {label}
    </button>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div
      className="flex items-center gap-2 text-sm py-8"
      style={{ color: "var(--muted-foreground)" }}
    >
      <Loader2 className="size-4 animate-spin" />
      {label}
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <p className="text-sm py-8" style={{ color: "var(--muted-foreground)" }}>
      {message}
    </p>
  );
}

function ErrorState({ message }: { message: string | null }) {
  return (
    <p className="text-sm py-8" style={{ color: "var(--destructive)" }}>
      {message ?? "Failed to load"}
    </p>
  );
}
