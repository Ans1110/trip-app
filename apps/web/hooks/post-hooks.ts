"use client";

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  postApi,
  type BookmarksQuery,
  type Comment,
  type CreateCommentInput,
  type CreatePostInput,
  type ListComments,
  type ListPosts,
  type ListPostsQuery,
  type ListReplies,
  type Post,
  type PostPageQuery,
  type UpdateCommentInput,
  type UpdatePostInput,
} from "@/lib/apis/post-api";
import { ApiError } from "@/lib/utils";

export const postKeys = {
  all: ["post"] as const,
  lists: () => [...postKeys.all, "list"] as const,
  list: (query?: ListPostsQuery) =>
    [
      ...postKeys.lists(),
      query?.author_id ?? null,
      query?.status ?? null,
      query?.cursor ?? null,
      query?.limit ?? null,
    ] as const,
  detail: (id: string) => [...postKeys.all, "detail", id] as const,
  comments: (postId: string, cursor?: string) =>
    [...postKeys.all, "comments", postId, cursor ?? null] as const,
  replies: (commentId: string, cursor?: string) =>
    [...postKeys.all, "replies", commentId, cursor ?? null] as const,
  bookmarks: () => [...postKeys.all, "bookmarks"] as const,
  bookmarkPage: (query?: BookmarksQuery) =>
    [
      ...postKeys.bookmarks(),
      query?.cursor ?? null,
      query?.limit ?? null,
    ] as const,
  infiniteList: (query?: Omit<ListPostsQuery, "cursor">) =>
    [
      ...postKeys.lists(),
      "infinite",
      query?.author_id ?? null,
      query?.status ?? null,
      query?.limit ?? null,
    ] as const,
  infiniteBookmarks: (limit?: number) =>
    [...postKeys.bookmarks(), "infinite", limit ?? null] as const,
};

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

export function usePosts(
  query?: ListPostsQuery,
  options?: QueryOpts<ListPosts>,
) {
  return useQuery<ListPosts, ApiError>({
    queryKey: postKeys.list(query),
    queryFn: ({ signal }) => postApi.list(query, signal),
    ...options,
  });
}

export function useInfinitePosts(query?: Omit<ListPostsQuery, "cursor">) {
  return useInfiniteQuery<ListPosts, ApiError>({
    queryKey: postKeys.infiniteList(query),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      postApi.list(
        { ...(query ?? {}), cursor: pageParam as string | undefined },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });
}

export function useInfiniteBookmarks(limit?: number) {
  return useInfiniteQuery<ListPosts, ApiError>({
    queryKey: postKeys.infiniteBookmarks(limit),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      postApi.listBookmarks(
        { cursor: pageParam as string | undefined, limit },
        signal,
      ),
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });
}

export function usePost(id: string, options?: QueryOpts<Post>) {
  return useQuery<Post, ApiError>({
    queryKey: postKeys.detail(id),
    queryFn: ({ signal }) => postApi.get(id, signal),
    enabled: id.length > 0,
    ...options,
  });
}

export function useCreatePost(options?: MutOpts<Post, CreatePostInput>) {
  const qc = useQueryClient();
  return useMutation<Post, ApiError, CreatePostInput>({
    mutationFn: (input) => postApi.create(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      qc.invalidateQueries({ queryKey: ["feed"] });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUpdatePost(
  id: string,
  options?: MutOpts<Post, UpdatePostInput>,
) {
  const qc = useQueryClient();
  return useMutation<Post, ApiError, UpdatePostInput>({
    mutationFn: (input) => postApi.update(id, input),
    ...options,
    onSuccess: (...args) => {
      const [data] = args;
      qc.setQueryData(postKeys.detail(id), data);
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useDeletePost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => postApi.remove(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.removeQueries({ queryKey: postKeys.detail(id) });
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      qc.invalidateQueries({ queryKey: ["feed"] });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useLikePost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => postApi.like(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: postKeys.detail(id) });
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUnlikePost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => postApi.unlike(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: postKeys.detail(id) });
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useBookmarkPost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => postApi.bookmark(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: postKeys.detail(id) });
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      qc.invalidateQueries({ queryKey: postKeys.bookmarks() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUnbookmarkPost(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => postApi.unbookmark(id),
    ...options,
    onSuccess: (...args) => {
      const [, id] = args;
      qc.invalidateQueries({ queryKey: postKeys.detail(id) });
      qc.invalidateQueries({ queryKey: postKeys.lists() });
      qc.invalidateQueries({ queryKey: postKeys.bookmarks() });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useBookmarks(
  query?: BookmarksQuery,
  options?: QueryOpts<ListPosts>,
) {
  return useQuery<ListPosts, ApiError>({
    queryKey: postKeys.bookmarkPage(query),
    queryFn: ({ signal }) => postApi.listBookmarks(query, signal),
    ...options,
  });
}

export function useComments(
  postId: string,
  query?: PostPageQuery,
  options?: QueryOpts<ListComments>,
) {
  return useQuery<ListComments, ApiError>({
    queryKey: postKeys.comments(postId, query?.cursor),
    queryFn: ({ signal }) => postApi.listComments(postId, query, signal),
    enabled: postId.length > 0,
    ...options,
  });
}

export function useCreateComment(
  postId: string,
  options?: MutOpts<Comment, CreateCommentInput>,
) {
  const qc = useQueryClient();
  return useMutation<Comment, ApiError, CreateCommentInput>({
    mutationFn: (input) => postApi.createComment(postId, input),
    ...options,
    onSuccess: (...args) => {
      const [data] = args;
      qc.invalidateQueries({ queryKey: [...postKeys.all, "comments", postId] });
      qc.invalidateQueries({ queryKey: postKeys.detail(postId) });
      // Response.parent_id is the resolved thread root; use the 3-element
      // prefix so every paginated page of that thread refetches.
      if (data.parent_id) {
        qc.invalidateQueries({
          queryKey: [...postKeys.all, "replies", data.parent_id],
        });
      }
      return options?.onSuccess?.(...args);
    },
  });
}

export function useUpdateComment(
  postId: string,
  commentId: string,
  options?: MutOpts<Comment, UpdateCommentInput>,
) {
  const qc = useQueryClient();
  return useMutation<Comment, ApiError, UpdateCommentInput>({
    mutationFn: (input) => postApi.updateComment(postId, commentId, input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: [...postKeys.all, "comments", postId] });
      return options?.onSuccess?.(...args);
    },
  });
}

export type DeleteCommentInput = { commentId: string; parentId?: string };

export function useDeleteComment(
  postId: string,
  options?: MutOpts<null, DeleteCommentInput>,
) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteCommentInput>({
    mutationFn: ({ commentId }) => postApi.deleteComment(postId, commentId),
    ...options,
    onSuccess: (...args) => {
      const [, input] = args;
      qc.invalidateQueries({ queryKey: [...postKeys.all, "comments", postId] });
      qc.invalidateQueries({ queryKey: postKeys.detail(postId) });
      if (input.parentId) {
        qc.invalidateQueries({
          queryKey: [...postKeys.all, "replies", input.parentId],
        });
      }
      return options?.onSuccess?.(...args);
    },
  });
}

export function useReplies(
  postId: string,
  commentId: string,
  query?: PostPageQuery,
  options?: QueryOpts<ListReplies>,
) {
  return useQuery<ListReplies, ApiError>({
    queryKey: postKeys.replies(commentId, query?.cursor),
    queryFn: ({ signal }) =>
      postApi.listReplies(postId, commentId, query, signal),
    enabled: postId.length > 0 && commentId.length > 0,
    ...options,
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
