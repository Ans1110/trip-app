import { z } from "zod";

export const creatableVisibilityEnum = z.enum(["private", "friends"]);
export const updatableVisibilityEnum = z.enum(["private", "friends", "public"]);
export const eventTypeEnum = z.enum([
  "general",
  "trip",
  "flight",
  "hotel",
  "meeting",
]);

const localDateTime = z.string().trim().min(1, "Required");

export const createEventSchema = z
  .object({
    title: z.string().trim().min(1, "Required").max(200, "Too long"),
    description: z.string().trim().max(2000, "Too long"),
    location: z.string().trim().max(200, "Too long"),
    start_at: localDateTime,
    end_at: localDateTime,
    all_day: z.boolean(),
    color: z.string().trim().max(16, "Too long"),
    event_type: eventTypeEnum,
    visibility: creatableVisibilityEnum,
  })
  .refine((d) => d.end_at >= d.start_at, {
    message: "End must be on or after start",
    path: ["end_at"],
  });
export type CreateEventFormInput = z.infer<typeof createEventSchema>;

export const updateEventSchema = z
  .object({
    title: z.string().trim().min(1, "Required").max(200, "Too long"),
    description: z.string().trim().max(2000, "Too long"),
    location: z.string().trim().max(200, "Too long"),
    start_at: localDateTime,
    end_at: localDateTime,
    all_day: z.boolean(),
    color: z.string().trim().max(16, "Too long"),
    event_type: eventTypeEnum,
    visibility: updatableVisibilityEnum,
  })
  .refine((d) => d.end_at >= d.start_at, {
    message: "End must be on or after start",
    path: ["end_at"],
  });
export type UpdateEventFormInput = z.infer<typeof updateEventSchema>;
