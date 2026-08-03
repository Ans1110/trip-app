import { z } from "zod";

const clearableUrl = z
  .string()
  .trim()
  .max(1024, "Too long")
  .url("Enter a valid URL")
  .or(z.literal(""))
  .optional();

const optionalSocialUrl = z
  .string()
  .trim()
  .max(256, "Too long")
  .url("Enter a valid URL")
  .or(z.literal(""))
  .optional()
  .transform((v) => (v === "" ? undefined : v));

export const updateProfileSchema = z.object({
  username: z
    .string()
    .trim()
    .min(3, "At least 3 characters")
    .max(32, "Too long")
    .regex(/^[a-zA-Z0-9_]+$/, "Letters, numbers, underscore only")
    .optional()
    .or(z.literal(""))
    .transform((v) => (v === "" ? undefined : v)),
  bio: z
    .string()
    .max(500, "Max 500 characters")
    .optional()
    .transform((v) => (v === "" ? undefined : v)),
  avatar_url: clearableUrl,
  cover_image: clearableUrl,
  travel_tags: z
    .array(z.string().trim().min(1).max(32))
    .max(20, "Up to 20 tags")
    .optional(),
  social_instagram: optionalSocialUrl,
  social_x: optionalSocialUrl,
  social_youtube: optionalSocialUrl,
  social_tiktok: optionalSocialUrl,
});

export type UpdateProfileFormInput = z.input<typeof updateProfileSchema>;
export type UpdateProfileFormOutput = z.output<typeof updateProfileSchema>;
