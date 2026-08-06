"use client";

import Link from "next/link";
import { useState } from "react";
import { Heart, Loader2, MessageCircle } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import { errorMessage, usePosts } from "@/hooks/post-hooks";
import type { Post, PostStatus } from "@/lib/apis/post-api";

type Tab = "published" | "draft" | "archived";

const TABS: { key: Tab; label: string }[] = [
  { key: "published", label: "Published" },
  { key: "draft", label: "Drafts" },
  { key: "archived", label: "Archived" },
];

export function UserPosts({ userId }: { userId: string }) {
  const { data: me } = useMe();
  const isOwn = me?.id === userId;
  const [tab, setTab] = useState<Tab>("published");
  const [cursor, setCursor] = useState<string | undefined>(undefined);

  const status: PostStatus = isOwn ? tab : "published";
  const { data, isLoading, error, isFetching } = usePosts({
    author_id: userId,
    status,
    cursor,
  });

  const selectTab = (next: Tab) => {
    if (next === tab) return;
    setTab(next);
    setCursor(undefined);
  };

  return (
    <section className="mt-10">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2
          className="text-lg font-semibold"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "#ECEFEA",
          }}
        >
          Posts
        </h2>
        {isOwn && (
          <div
            className="inline-flex items-center gap-1 p-1 rounded-full"
            style={{ border: "1px solid #1F2A24" }}
          >
            {TABS.map((t) => {
              const active = t.key === tab;
              return (
                <button
                  key={t.key}
                  type="button"
                  onClick={() => selectTab(t.key)}
                  aria-pressed={active}
                  className="text-xs px-3 py-1.5 rounded-full season-transition"
                  style={{
                    backgroundColor: active
                      ? "color-mix(in srgb, var(--season-accent) 18%, transparent)"
                      : "transparent",
                    color: active ? "var(--season-accent)" : "#8B9A8E",
                  }}
                >
                  {t.label}
                </button>
              );
            })}
          </div>
        )}
      </div>

      {isLoading ? (
        <div
          className="flex items-center gap-2 text-sm"
          style={{ color: "#8B9A8E" }}
        >
          <Loader2 className="size-4 animate-spin" />
          Loading posts
        </div>
      ) : error ? (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(error) ?? "Failed to load posts"}
        </p>
      ) : !data || data.posts.length === 0 ? (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {isOwn && tab !== "published" ? `No ${tab} posts.` : "No posts yet."}
        </p>
      ) : (
        <>
          <ul className="flex flex-col gap-3">
            {data.posts.map((p) => (
              <PostRow key={p.id} post={p} />
            ))}
          </ul>
          {data.next_cursor && data.posts.length > 3 && (
            <div className="mt-6 flex justify-center">
              <button
                type="button"
                onClick={() => setCursor(data.next_cursor)}
                disabled={isFetching}
                className="px-4 py-2 text-sm rounded-full border hover:bg-white/5 disabled:opacity-60"
                style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
              >
                {isFetching ? "Loading…" : "Load more"}
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function PostRow({ post }: { post: Post }) {
  return (
    <li
      className="flex gap-4 p-4 rounded-xl border"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      {post.cover_image && (
        <Link
          href={`/posts/${post.id}`}
          className="size-16 rounded-lg overflow-hidden shrink-0"
          aria-hidden
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={post.cover_image}
            alt=""
            className="size-full object-cover"
          />
        </Link>
      )}
      <div className="flex-1 min-w-0">
        <Link
          href={`/posts/${post.id}`}
          className="block font-medium hover:underline truncate"
          style={{ color: "#ECEFEA" }}
        >
          {post.title}
        </Link>
        {post.tags.length > 0 && (
          <ul className="mt-1.5 flex flex-wrap gap-1">
            {post.tags.slice(0, 6).map((tag) => (
              <li
                key={tag}
                className="text-[11px] px-2 py-0.5 rounded-full"
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
          className="mt-2 flex items-center gap-4 text-xs"
          style={{ color: "#8B9A8E" }}
        >
          <span className="inline-flex items-center gap-1">
            <Heart className="size-3.5" /> {post.like_count}
          </span>
          <span className="inline-flex items-center gap-1">
            <MessageCircle className="size-3.5" /> {post.comment_count}
          </span>
          <span style={{ color: "#6B7A6F" }}>
            {formatWhen(post.published_at)}
          </span>
        </div>
      </div>
    </li>
  );
}

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString();
}
