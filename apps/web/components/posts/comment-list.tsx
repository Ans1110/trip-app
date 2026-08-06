"use client";

import Link from "next/link";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { CornerDownRight, Loader2, MessageSquare, Trash2 } from "lucide-react";

import {
  errorMessage,
  useComments,
  useCreateComment,
  useDeleteComment,
  useReplies,
} from "@/hooks/post-hooks";
import type { Comment } from "@/lib/apis/post-api";
import {
  commentSchema,
  type CommentFormInput,
} from "@/lib/schemas/post-schemas";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";

export function CommentList({ postId }: { postId: string }) {
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const { data, isLoading, error } = useComments(postId, { cursor });

  return (
    <section className="mt-10">
      <h2
        className="text-lg font-semibold mb-4"
        style={{
          fontFamily: "var(--font-display, Georgia, serif)",
          color: "#ECEFEA",
        }}
      >
        Comments
      </h2>

      <CommentComposer postId={postId} />

      {isLoading ? (
        <div
          className="mt-6 flex items-center gap-2 text-sm"
          style={{ color: "#8B9A8E" }}
        >
          <Loader2 className="size-4 animate-spin" />
          Loading comments
        </div>
      ) : error ? (
        <p className="mt-6 text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(error) ?? "Failed to load comments"}
        </p>
      ) : !data || data.comments.length === 0 ? (
        <p className="mt-6 text-sm" style={{ color: "#8B9A8E" }}>
          Be the first to comment.
        </p>
      ) : (
        <>
          <ul className="mt-6 flex flex-col gap-3">
            {data.comments.map((c) => (
              <CommentRow key={c.id} postId={postId} comment={c} />
            ))}
          </ul>
          {data.next_cursor && data.comments.length > 5 && (
            <div className="mt-4 flex justify-center">
              <button
                type="button"
                onClick={() => setCursor(data.next_cursor)}
                className="px-4 py-2 text-sm rounded-full border hover:bg-white/5"
                style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
              >
                Load more
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function CommentComposer({
  postId,
  parentId,
  autoFocus,
  onDone,
  placeholder = "Share a thought…",
  submitLabel = "Comment",
}: {
  postId: string;
  parentId?: string;
  autoFocus?: boolean;
  onDone?: () => void;
  placeholder?: string;
  submitLabel?: string;
}) {
  const create = useCreateComment(postId);
  const form = useForm<CommentFormInput>({
    resolver: zodResolver(commentSchema),
    defaultValues: { content: "" },
  });

  const onSubmit = (v: CommentFormInput) => {
    create.mutate(
      { content: v.content.trim(), parent_id: parentId },
      {
        onSuccess: () => {
          form.reset({ content: "" });
          onDone?.();
        },
      },
    );
  };

  const submitError = errorMessage(create.error);

  return (
    <FormProvider {...form}>
      <form
        className="flex flex-col gap-2"
        onSubmit={form.handleSubmit(onSubmit)}
        noValidate
      >
        <ControlledTextarea<CommentFormInput>
          name="content"
          placeholder={placeholder}
          rows={parentId ? 2 : 3}
          autoFocus={autoFocus}
        />
        {submitError && (
          <p className="text-xs" style={{ color: "#FCA5A5" }}>
            {submitError}
          </p>
        )}
        <div className="flex justify-end gap-2">
          {onDone && (
            <button
              type="button"
              onClick={onDone}
              className="px-3 py-1.5 text-xs rounded-full border hover:bg-white/5"
              style={{ borderColor: "#1F2A24", color: "#8B9A8E" }}
            >
              Cancel
            </button>
          )}
          <button
            type="submit"
            disabled={create.isPending}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {create.isPending && <Loader2 className="size-3.5 animate-spin" />}
            {submitLabel}
          </button>
        </div>
      </form>
    </FormProvider>
  );
}

function CommentRow({
  postId,
  comment,
  replyTargetName,
  nestedReplies,
}: {
  postId: string;
  comment: Comment;
  replyTargetName?: string;
  nestedReplies?: React.ReactNode;
}) {
  const del = useDeleteComment(postId);
  const [replyOpen, setReplyOpen] = useState(false);
  const [threadOpen, setThreadOpen] = useState(false);
  const isReply = !!comment.parent_id;

  return (
    <li
      className="flex flex-col gap-2 p-3 rounded-xl border"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      <CommentBody
        comment={comment}
        replyTargetName={replyTargetName}
        onDelete={
          comment.is_author
            ? () =>
                del.mutate({
                  commentId: comment.id,
                  parentId: comment.parent_id,
                })
            : undefined
        }
        deleting={del.isPending}
      />

      {!comment.is_deleted && (
        <div className="flex items-center gap-3 pl-11 text-xs">
          <button
            type="button"
            onClick={() => setReplyOpen((v) => !v)}
            className="inline-flex items-center gap-1 hover:underline"
            style={{ color: "#8B9A8E" }}
          >
            <CornerDownRight className="size-3.5" />
            Reply
          </button>
          {!isReply && comment.reply_count > 0 && (
            <button
              type="button"
              onClick={() => setThreadOpen((v) => !v)}
              className="inline-flex items-center gap-1 hover:underline"
              style={{ color: "var(--season-accent)" }}
            >
              <MessageSquare className="size-3.5" />
              {threadOpen ? "Hide" : "View"} {comment.reply_count}{" "}
              {comment.reply_count === 1 ? "reply" : "replies"}
            </button>
          )}
        </div>
      )}

      {replyOpen && (
        <div className="pl-11">
          <CommentComposer
            postId={postId}
            parentId={comment.id}
            autoFocus
            placeholder={`Reply to ${comment.author.name || "@" + (comment.author.username ?? "user")}…`}
            submitLabel="Reply"
            onDone={() => {
              setReplyOpen(false);
              if (!isReply) setThreadOpen(true);
            }}
          />
        </div>
      )}

      {nestedReplies}

      {threadOpen && !isReply && (
        <ReplyThread postId={postId} parentId={comment.id} />
      )}
    </li>
  );
}

function CommentBody({
  comment,
  onDelete,
  deleting,
  replyTargetName,
}: {
  comment: Comment;
  onDelete?: () => void;
  deleting?: boolean;
  replyTargetName?: string;
}) {
  const author = comment.author;
  const initial = (author.username || author.name || "U")
    .trim()[0]!
    .toUpperCase();
  const authorHref = author.username
    ? `/profile/${author.username}`
    : `/profile/${author.user_id}`;
  const displayName =
    author.name || (author.username ? `@${author.username}` : "user");

  return (
    <div className="flex gap-3">
      <Link
        href={authorHref}
        className="size-8 rounded-full overflow-hidden inline-flex items-center justify-center text-xs font-semibold shrink-0"
        style={{
          backgroundColor: author.avatar_url
            ? "transparent"
            : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
          color: "#ECEFEA",
          opacity: comment.is_deleted ? 0.5 : 1,
        }}
      >
        {author.avatar_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={author.avatar_url}
            alt={author.name}
            className="size-full object-cover"
          />
        ) : (
          <span aria-hidden>{initial}</span>
        )}
      </Link>
      <div className="flex-1 min-w-0">
        <div className="flex items-baseline justify-between gap-2">
          <Link
            href={authorHref}
            className="text-sm font-semibold hover:underline truncate"
            style={{
              color: "#ECEFEA",
              opacity: comment.is_deleted ? 0.5 : 1,
            }}
          >
            {displayName}
          </Link>
          <span className="text-[11px] shrink-0" style={{ color: "#6B7A6F" }}>
            {formatWhen(comment.created_at)}
          </span>
        </div>
        {comment.is_deleted ? (
          <p className="mt-1 text-sm italic" style={{ color: "#6B7A6F" }}>
            [deleted]
          </p>
        ) : (
          <>
            {replyTargetName && (
              <p
                className="mt-1 text-[11px]"
                style={{ color: "var(--season-accent)" }}
              >
                Replying to {replyTargetName}
              </p>
            )}
            <p
              className="mt-1 text-sm whitespace-pre-wrap"
              style={{ color: "#ECEFEA" }}
            >
              {comment.content}
            </p>
          </>
        )}
      </div>
      {onDelete && (
        <button
          type="button"
          onClick={onDelete}
          disabled={deleting}
          aria-label="Delete comment"
          className="self-start inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60"
          style={{ color: "#8B9A8E" }}
        >
          {deleting ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Trash2 className="size-3.5" />
          )}
        </button>
      )}
    </div>
  );
}

function ReplyThread({
  postId,
  parentId,
}: {
  postId: string;
  parentId: string;
}) {
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const { data, isLoading, error } = useReplies(postId, parentId, { cursor });

  if (isLoading) {
    return (
      <div
        className="pl-11 flex items-center gap-2 text-xs"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-3.5 animate-spin" />
        Loading replies
      </div>
    );
  }
  if (error) {
    return (
      <p className="pl-11 text-xs" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Failed to load replies"}
      </p>
    );
  }
  if (!data || data.replies.length === 0) {
    return null;
  }

  const forest = buildReplyForest(data.replies, parentId);
  const byId = new Map(data.replies.map((r) => [r.id, r] as const));

  return (
    <div className="pl-11 flex flex-col gap-2">
      <ul
        className="flex flex-col gap-2 border-l pl-3"
        style={{ borderColor: "#1F2A24" }}
      >
        {forest.map((node) => (
          <ReplyBranch
            key={node.comment.id}
            node={node}
            postId={postId}
            rootId={parentId}
            byId={byId}
          />
        ))}
      </ul>
      {data.next_cursor && data.replies.length >= 20 && (
        <div className="flex justify-start">
          <button
            type="button"
            onClick={() => setCursor(data.next_cursor)}
            className="text-xs hover:underline"
            style={{ color: "var(--season-accent)" }}
          >
            Load more replies
          </button>
        </div>
      )}
    </div>
  );
}

type ReplyNode = { comment: Comment; children: ReplyNode[] };

// Backend flattens all replies onto the same parent_id (thread root), but
// in_reply_to_id preserves who was actually replied to. Rebuild that shape
// as a nested forest so replies-to-replies can render indented under their
// sibling target instead of at the root level.
function buildReplyForest(replies: Comment[], rootId: string): ReplyNode[] {
  const nodes = new Map<string, ReplyNode>();
  for (const r of replies) nodes.set(r.id, { comment: r, children: [] });
  const roots: ReplyNode[] = [];
  for (const r of replies) {
    const node = nodes.get(r.id)!;
    const targetId = r.in_reply_to_id;
    const parent =
      targetId && targetId !== rootId ? nodes.get(targetId) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  return roots;
}

function ReplyBranch({
  node,
  postId,
  rootId,
  byId,
}: {
  node: ReplyNode;
  postId: string;
  rootId: string;
  byId: Map<string, Comment>;
}) {
  const targetId = node.comment.in_reply_to_id;
  const target =
    targetId && targetId !== rootId ? byId.get(targetId) : undefined;
  const targetName = target
    ? target.author.name ||
      (target.author.username ? `@${target.author.username}` : undefined)
    : undefined;
  return (
    <CommentRow
      postId={postId}
      comment={node.comment}
      replyTargetName={targetName}
      nestedReplies={
        node.children.length > 0 ? (
          <ul
            className="ml-4 mt-1 flex flex-col gap-2 border-l pl-3"
            style={{ borderColor: "#1F2A24" }}
          >
            {node.children.map((c) => (
              <ReplyBranch
                key={c.comment.id}
                node={c}
                postId={postId}
                rootId={rootId}
                byId={byId}
              />
            ))}
          </ul>
        ) : null
      }
    />
  );
}

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
