import { z } from "zod";

// ---- Trip ----

export const createTripSchema = z
  .object({
    title: z.string().trim().min(1, "Required").max(120, "Too long"),
    description: z.string().trim().max(1000, "Too long"),
    start_date: z.string().min(1, "Required"),
    end_date: z.string().min(1, "Required"),
  })
  .refine((d) => d.end_date >= d.start_date, {
    message: "End date must be on or after start date",
    path: ["end_date"],
  });
export type CreateTripFormInput = z.infer<typeof createTripSchema>;

// ---- Room ----

export const joinRoomSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, "Required")
    .max(16, "Too long")
    .transform((s) => s.toUpperCase()),
});
export type JoinRoomFormInput = z.infer<typeof joinRoomSchema>;

// ---- Itinerary ----

export const createItinerarySchema = z.object({
  day: z
    .number({ invalid_type_error: "Required" })
    .int("Whole number")
    .min(1, "Min 1"),
  title: z.string().trim().max(120, "Too long"),
  location: z.string().trim().max(120, "Too long"),
});
export type CreateItineraryFormInput = z.infer<typeof createItinerarySchema>;

const optionalCoord = (min: number, max: number, label: string) =>
  z
    .string()
    .trim()
    .refine(
      (s) => {
        if (s === "") return true;
        const n = Number(s);
        return Number.isFinite(n) && n >= min && n <= max;
      },
      { message: label },
    );

export const updateItinerarySchema = z.object({
  title: z.string().trim().max(120, "Too long"),
  location: z.string().trim().max(120, "Too long"),
  description: z.string().trim().max(1000, "Too long"),
  latitude: optionalCoord(-90, 90, "Latitude must be between -90 and 90"),
  longitude: optionalCoord(-180, 180, "Longitude must be between -180 and 180"),
});
export type UpdateItineraryFormInput = z.infer<typeof updateItinerarySchema>;

// ---- Todos ----

export const todoPriorityEnum = z.enum(["low", "normal", "high"]);

export const createTodoSchema = z.object({
  title: z.string().trim().min(1, "Required").max(200, "Too long"),
  priority: todoPriorityEnum,
  tags: z.array(z.string().trim().min(1).max(30)).max(10, "Too many tags"),
});
export type CreateTodoFormInput = z.infer<typeof createTodoSchema>;
