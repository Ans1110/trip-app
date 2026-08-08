import { z } from "zod";

export const elevateSchema = z.object({
  password: z.string().min(1, "Required").max(128, "Too long"),
});
export type ElevateInput = z.infer<typeof elevateSchema>;

export const createReportSchema = z.object({
  subject_type: z.enum(["post", "comment", "user"]),
  subject_id: z.string().uuid("Invalid id"),
  reason: z.string().trim().min(1, "Required").max(32, "Too long"),
  description: z.string().trim().max(1000, "Too long").optional(),
});
export type CreateReportInput = z.infer<typeof createReportSchema>;

export const resolveReportSchema = z.object({
  resolution: z.string().trim().max(1000, "Too long").optional(),
});
export type ResolveReportInput = z.infer<typeof resolveReportSchema>;
