import { z } from "zod";

export const POST_TITLE_MAX = 200;
export const POST_CONTENT_MAX = 20000;
export const POST_COVER_URL_MAX = 1024;
export const POST_TAGS_MAX = 20;
export const POST_TAG_MAX = 32;
export const COMMENT_MAX = 2000;

const optionalUrl = z
  .string()
  .trim()
  .max(POST_COVER_URL_MAX, "Too long")
  .url("Enter a valid URL")
  .or(z.literal(""))
  .optional()
  .transform((v) => (v === "" || v === undefined ? undefined : v));

const tagsField = z
  .array(z.string().trim().min(1).max(POST_TAG_MAX))
  .max(POST_TAGS_MAX, `Up to ${POST_TAGS_MAX} tags`)
  .optional();

const statusField = z.enum(["draft", "published", "archived"]).optional();

export const createPostSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, "Title is required")
    .max(POST_TITLE_MAX, `Max ${POST_TITLE_MAX} characters`),
  content: z
    .string()
    .min(1, "Content is required")
    .max(POST_CONTENT_MAX, `Max ${POST_CONTENT_MAX} characters`),
  cover_image: optionalUrl,
  tags: tagsField,
  status: statusField,
});

export const updatePostSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, "Title is required")
    .max(POST_TITLE_MAX, `Max ${POST_TITLE_MAX} characters`)
    .optional(),
  content: z
    .string()
    .min(1, "Content is required")
    .max(POST_CONTENT_MAX, `Max ${POST_CONTENT_MAX} characters`)
    .optional(),
  cover_image: optionalUrl,
  tags: tagsField,
  status: statusField,
});

export const commentSchema = z.object({
  content: z
    .string()
    .trim()
    .min(1, "Comment cannot be empty")
    .max(COMMENT_MAX, `Max ${COMMENT_MAX} characters`),
});

export type CreatePostFormInput = z.input<typeof createPostSchema>;
export type CreatePostFormOutput = z.output<typeof createPostSchema>;
export type UpdatePostFormInput = z.input<typeof updatePostSchema>;
export type UpdatePostFormOutput = z.output<typeof updatePostSchema>;
export type CommentFormInput = z.input<typeof commentSchema>;
export type CommentFormOutput = z.output<typeof commentSchema>;
