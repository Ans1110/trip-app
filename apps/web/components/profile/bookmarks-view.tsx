"use client";

import Link from "next/link";
import { useState } from "react";
import { Heart, Loader2, MessageCircle } from "lucide-react";

import { errorMessage, useBookmarks } from "@/hooks/post-hooks";
import type { Post } from "@/lib/apis/post-api";

export function BookmarksView() {
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const { data, isLoading, error, isFetching } = useBookmarks({ cursor });

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading bookmarks
      </div>
    );
  }
  if (error) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Failed to load bookmarks"}
      </p>
    );
  }
  if (!data || data.posts.length === 0) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        No bookmarks yet — tap Save on any post to keep it here.
      </p>
    );
  }

  return (
    <div>
      <ul className="flex flex-col gap-3">
        {data.posts.map((post) => (
          <BookmarkRow key={post.id} post={post} />
        ))}
      </ul>
      {data.next_cursor && (
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
    </div>
  );
}

function BookmarkRow({ post }: { post: Post }) {
  const author = post.author;
  const authorHref = author.username
    ? `/profile/${author.username}`
    : `/profile/${author.user_id}`;
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
        <p className="mt-0.5 text-xs truncate" style={{ color: "#8B9A8E" }}>
          by{" "}
          <Link href={authorHref} className="hover:underline">
            {author.name || (author.username ? `@${author.username}` : "user")}
          </Link>
        </p>
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
        </div>
      </div>
    </li>
  );
}
