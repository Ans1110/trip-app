"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Bookmark,
  Heart,
  Loader2,
  MessageCircle,
  Pencil,
  Trash2,
} from "lucide-react";

import {
  errorMessage,
  useBookmarkPost,
  useDeletePost,
  useLikePost,
  usePost,
  useUnbookmarkPost,
  useUnlikePost,
} from "@/hooks/post-hooks";
import type { PostStatus } from "@/lib/apis/post-api";
import { CommentList } from "./comment-list";

export function PostDetail({ postId }: { postId: string }) {
  const router = useRouter();
  const { data: post, isLoading, error } = usePost(postId);
  const like = useLikePost();
  const unlike = useUnlikePost();
  const bookmark = useBookmarkPost();
  const unbookmark = useUnbookmarkPost();
  const del = useDeletePost();

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading post
      </div>
    );
  }
  if (error || !post) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Post not found"}
      </p>
    );
  }

  const author = post.author;
  const initial = (author.username || author.name || "U")
    .trim()[0]!
    .toUpperCase();
  const authorHref = author.username
    ? `/profile/${author.username}`
    : `/profile/${author.user_id}`;
  const likePending = like.isPending || unlike.isPending;
  const bookmarkPending = bookmark.isPending || unbookmark.isPending;

  const toggleLike = () => {
    if (post.is_liked) unlike.mutate(post.id);
    else like.mutate(post.id);
  };

  const toggleBookmark = () => {
    if (post.is_bookmarked) unbookmark.mutate(post.id);
    else bookmark.mutate(post.id);
  };

  const onDelete = () => {
    if (!window.confirm("Delete this post? This cannot be undone.")) return;
    del.mutate(post.id, {
      onSuccess: () => router.push("/feed"),
    });
  };

  return (
    <article>
      {post.cover_image && (
        <div
          className="mb-6 rounded-2xl overflow-hidden"
          style={{ border: "1px solid #1F2A24" }}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={post.cover_image}
            alt=""
            className="w-full max-h-[420px] object-cover"
          />
        </div>
      )}

      <header className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h1
            className="leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.75rem, 3vw, 2.5rem)",
              color: "var(--season-heading, #ECEFEA)",
            }}
          >
            {post.title}
          </h1>
          {post.status !== "published" && <StatusBadge status={post.status} />}
          <div className="mt-3 flex items-center gap-3">
            <Link
              href={authorHref}
              className="size-9 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
              style={{
                backgroundColor: author.avatar_url
                  ? "transparent"
                  : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
                color: "#ECEFEA",
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
            <div className="min-w-0">
              <Link
                href={authorHref}
                className="text-sm font-semibold hover:underline"
                style={{ color: "#ECEFEA" }}
              >
                {author.name ||
                  (author.username ? `@${author.username}` : "user")}
              </Link>
              <p className="text-xs" style={{ color: "#6B7A6F" }}>
                {formatWhen(post.published_at)}
              </p>
            </div>
          </div>
        </div>

        {post.is_author && (
          <div className="flex items-center gap-2 shrink-0">
            <Link
              href={`/posts/${post.id}/edit`}
              className="inline-flex items-center gap-1 text-xs px-3 py-1.5 rounded-full border hover:bg-white/5"
              style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
            >
              <Pencil className="size-3.5" />
              Edit
            </Link>
            <button
              type="button"
              onClick={onDelete}
              disabled={del.isPending}
              className="inline-flex items-center gap-1 text-xs px-3 py-1.5 rounded-full border hover:bg-white/5 disabled:opacity-60"
              style={{ borderColor: "#1F2A24", color: "#FCA5A5" }}
            >
              {del.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
              Delete
            </button>
          </div>
        )}
      </header>

      {post.tags.length > 0 && (
        <ul className="mt-4 flex flex-wrap gap-1.5">
          {post.tags.map((tag) => (
            <li
              key={tag}
              className="text-xs px-2.5 py-1 rounded-full"
              style={{
                backgroundColor:
                  "color-mix(in srgb, var(--season-accent) 14%, transparent)",
                color: "var(--season-accent)",
              }}
            >
              {tag}
            </li>
          ))}
        </ul>
      )}

      <div
        className="mt-6 text-[15px] leading-relaxed whitespace-pre-wrap"
        style={{ color: "#ECEFEA" }}
      >
        {post.content}
      </div>

      <div className="mt-8 flex items-center gap-4">
        <button
          type="button"
          onClick={toggleLike}
          disabled={likePending}
          aria-pressed={post.is_liked}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-full border text-sm hover:bg-white/5 disabled:opacity-60"
          style={{
            borderColor: post.is_liked ? "var(--season-accent)" : "#1F2A24",
            color: post.is_liked ? "var(--season-accent)" : "#ECEFEA",
          }}
        >
          <Heart
            className="size-4"
            fill={post.is_liked ? "currentColor" : "none"}
          />
          {post.like_count}
        </button>
        <button
          type="button"
          onClick={toggleBookmark}
          disabled={bookmarkPending}
          aria-pressed={post.is_bookmarked}
          aria-label={post.is_bookmarked ? "Remove bookmark" : "Bookmark"}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-full border text-sm hover:bg-white/5 disabled:opacity-60"
          style={{
            borderColor: post.is_bookmarked
              ? "var(--season-accent)"
              : "#1F2A24",
            color: post.is_bookmarked ? "var(--season-accent)" : "#ECEFEA",
          }}
        >
          <Bookmark
            className="size-4"
            fill={post.is_bookmarked ? "currentColor" : "none"}
          />
          {post.is_bookmarked ? "Saved" : "Save"}
        </button>
        <span
          className="inline-flex items-center gap-2 text-sm"
          style={{ color: "#8B9A8E" }}
        >
          <MessageCircle className="size-4" />
          {post.comment_count}
        </span>
      </div>

      <CommentList postId={post.id} />
    </article>
  );
}

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function StatusBadge({ status }: { status: PostStatus }) {
  const label = status === "draft" ? "Draft" : "Archived";
  return (
    <span
      className="mt-2 inline-flex items-center text-[11px] px-2 py-0.5 rounded-full"
      style={{
        backgroundColor: "color-mix(in srgb, #8B9A8E 18%, transparent)",
        color: "#ECEFEA",
        border: "1px solid #1F2A24",
      }}
    >
      {label}
    </span>
  );
}
