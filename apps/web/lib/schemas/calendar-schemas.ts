import { z } from "zod";

export const creatableVisibilityEnum = z.enum(["private", "friends"]);
export const updatableVisibilityEnum = z.enum(["private", "friends"]);

const dateStr = z.string().trim().min(1, "Required");
const timeStr = z
  .string()
  .trim()
  .regex(/^\d{2}:\d{2}$/, "Invalid time");

const combineDT = (d: string, t: string): string => `${d}T${t}`;

export const createEventSchema = z
  .object({
    title: z.string().trim().min(1, "Required").max(200, "Too long"),
    description: z.string().trim().max(2000, "Too long"),
    location: z.string().trim().max(200, "Too long"),
    start_date: dateStr,
    start_time: timeStr,
    end_date: dateStr,
    end_time: timeStr,
    all_day: z.boolean(),
    color: z.string().trim().max(16, "Too long"),
    visibility: creatableVisibilityEnum,
  })
  .refine(
    (d) =>
      combineDT(d.end_date, d.end_time) >=
      combineDT(d.start_date, d.start_time),
    { message: "End must be on or after start", path: ["end_date"] },
  );
export type CreateEventFormInput = z.infer<typeof createEventSchema>;

export const updateEventSchema = z
  .object({
    title: z.string().trim().min(1, "Required").max(200, "Too long"),
    description: z.string().trim().max(2000, "Too long"),
    location: z.string().trim().max(200, "Too long"),
    start_date: dateStr,
    start_time: timeStr,
    end_date: dateStr,
    end_time: timeStr,
    all_day: z.boolean(),
    color: z.string().trim().max(16, "Too long"),
    visibility: updatableVisibilityEnum,
  })
  .refine(
    (d) =>
      combineDT(d.end_date, d.end_time) >=
      combineDT(d.start_date, d.start_time),
    { message: "End must be on or after start", path: ["end_date"] },
  );
export type UpdateEventFormInput = z.infer<typeof updateEventSchema>;
