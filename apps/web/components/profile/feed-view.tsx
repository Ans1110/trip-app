"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef } from "react";
import { ChevronRight, Heart, Loader2, MessageCircle } from "lucide-react";

import type { Post } from "@/lib/apis/post-api";
import { errorMessage, useInfiniteFeed } from "@/hooks/feed-hooks";
import { useInfiniteBookmarks, useInfinitePosts } from "@/hooks/post-hooks";
import { useInfiniteSearchPosts } from "@/hooks/search-hooks";

export type FeedTab = "all" | "following" | "bookmarks";

export function FeedView({
  query,
  tab = "following",
}: {
  query?: string;
  tab?: FeedTab;
}) {
  const q = (query ?? "").trim();
  if (q.length > 0) return <SearchResults key={q} query={q} />;
  if (tab === "all") return <AllFeed key="all" />;
  if (tab === "bookmarks") return <BookmarksFeed key="bookmarks" />;
  return <FollowingFeed key="following" />;
}

type InfiniteData<T extends { posts: Post[]; next_cursor?: string }> = {
  pages: T[];
};

const flatten = <T extends { posts: Post[] }>(data?: InfiniteData<T>): Post[] =>
  data?.pages.flatMap((p) => p.posts) ?? [];

function AllFeed() {
  const q = useInfinitePosts({ status: "published" });
  return (
    <PostFeedList
      posts={useMemo(() => flatten(q.data), [q.data])}
      isLoading={q.isLoading}
      isFetchingNextPage={q.isFetchingNextPage}
      hasNextPage={q.hasNextPage}
      fetchNextPage={q.fetchNextPage}
      error={q.error}
      emptyMessage="No posts yet."
    />
  );
}

function FollowingFeed() {
  const q = useInfiniteFeed();
  return (
    <PostFeedList
      posts={useMemo(() => flatten(q.data), [q.data])}
      isLoading={q.isLoading}
      isFetchingNextPage={q.isFetchingNextPage}
      hasNextPage={q.hasNextPage}
      fetchNextPage={q.fetchNextPage}
      error={q.error}
      emptyMessage="No posts yet — check back after some travelers publish their trips."
    />
  );
}

function BookmarksFeed() {
  const q = useInfiniteBookmarks();
  return (
    <PostFeedList
      posts={useMemo(() => flatten(q.data), [q.data])}
      isLoading={q.isLoading}
      isFetchingNextPage={q.isFetchingNextPage}
      hasNextPage={q.hasNextPage}
      fetchNextPage={q.fetchNextPage}
      error={q.error}
      emptyMessage="No bookmarks yet — tap Save on any post to keep it here."
    />
  );
}

function SearchResults({ query }: { query: string }) {
  const q = useInfiniteSearchPosts(query);
  return (
    <PostFeedList
      posts={useMemo(() => flatten(q.data), [q.data])}
      isLoading={q.isLoading}
      isFetchingNextPage={q.isFetchingNextPage}
      hasNextPage={q.hasNextPage}
      fetchNextPage={q.fetchNextPage}
      error={q.error}
      loadingLabel="Searching"
      emptyMessage={`No posts match “${query}”.`}
    />
  );
}

function PostFeedList({
  posts,
  isLoading,
  isFetchingNextPage,
  hasNextPage,
  fetchNextPage,
  error,
  emptyMessage,
  loadingLabel = "Loading feed",
}: {
  posts: Post[];
  isLoading: boolean;
  isFetchingNextPage: boolean;
  hasNextPage: boolean | undefined;
  fetchNextPage: () => void;
  error: unknown;
  emptyMessage: string;
  loadingLabel?: string;
}) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm" style={{ color: "#8B9A8E" }}>
        <Loader2 className="size-4 animate-spin" />
        {loadingLabel}
      </div>
    );
  }
  if (error) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Failed to load"}
      </p>
    );
  }
  if (posts.length === 0) {
    return (
      <p className="text-sm" style={{ color: "#8B9A8E" }}>
        {emptyMessage}
      </p>
    );
  }

  return (
    <div>
      <ul className="flex flex-col gap-3">
        {posts.map((post) => (
          <FeedRow key={post.id} post={post} />
        ))}
      </ul>
      <InfiniteSentinel
        hasNextPage={!!hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
        onLoadMore={fetchNextPage}
      />
    </div>
  );
}

function InfiniteSentinel({
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
}: {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
}) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!hasNextPage) return;
    const node = ref.current;
    if (!node) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting) && !isFetchingNextPage) {
          onLoadMore();
        }
      },
      { rootMargin: "400px 0px" },
    );
    io.observe(node);
    return () => io.disconnect();
  }, [hasNextPage, isFetchingNextPage, onLoadMore]);

  if (!hasNextPage) return null;
  return (
    <div
      ref={ref}
      className="mt-6 flex items-center justify-center gap-2 py-4 text-xs"
      style={{ color: "#8B9A8E" }}
      aria-hidden={!isFetchingNextPage}
    >
      {isFetchingNextPage && (
        <>
          <Loader2 className="size-3.5 animate-spin" />
          Loading more
        </>
      )}
    </div>
  );
}

function FeedRow({ post }: { post: Post }) {
  const author = post.author;
  const authorInitial =
    (author.username || author.name || "U").trim()[0]!.toUpperCase();
  const authorHref = author.username
    ? `/profile/${author.username}`
    : `/profile/${author.user_id}`;
  const displayTags = Array.from(
    new Set(post.tags.map((t) => t.trim()).filter((t) => t.length > 0)),
  ).slice(0, 6);

  return (
    <li
      className="flex gap-4 p-4 rounded-xl border"
      style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
    >
      <Link
        href={authorHref}
        className="size-10 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
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
          <span aria-hidden>{authorInitial}</span>
        )}
      </Link>
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
          <Link href={authorHref} className="font-semibold hover:underline">
            {author.name || (author.username ? `@${author.username}` : "user")}
          </Link>{" "}
          <span style={{ color: "#8B9A8E" }}>published a post</span>
        </p>
        <Link
          href={`/posts/${post.id}`}
          className="mt-1 block font-medium hover:underline"
          style={{ color: "#ECEFEA" }}
        >
          {post.title}
        </Link>
        {displayTags.length > 0 && (
          <ul className="mt-2 flex flex-wrap gap-1">
            {displayTags.map((tag) => (
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
      <Link
        href={`/posts/${post.id}`}
        className="self-center inline-flex items-center gap-1 text-xs px-3 py-1.5 rounded-full border hover:bg-white/5"
        style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
      >
        View
        <ChevronRight className="size-3.5" />
      </Link>
    </li>
  );
}

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
