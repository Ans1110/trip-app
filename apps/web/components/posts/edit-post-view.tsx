"use client";

import { Loader2 } from "lucide-react";

import { errorMessage, usePost } from "@/hooks/post-hooks";
import { PostForm } from "./post-form";

export function EditPostView({ postId }: { postId: string }) {
  const { data: post, isLoading, error } = usePost(postId);

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm"
        style={{ color: "var(--muted-foreground)" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading post
      </div>
    );
  }
  if (error || !post) {
    return (
      <p className="text-sm" style={{ color: "var(--destructive)" }}>
        {errorMessage(error) ?? "Post not found"}
      </p>
    );
  }
  if (!post.is_author) {
    return (
      <p className="text-sm" style={{ color: "var(--destructive)" }}>
        You cannot edit this post.
      </p>
    );
  }

  return <PostForm mode="edit" initial={post} />;
}
