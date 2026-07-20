import { z } from "zod";

export const pollTypeEnum = z.enum(["location", "time", "custom"]);
export const pollVisibilityEnum = z.enum([
  "always",
  "after_vote",
  "after_deadline",
]);

// createPollSchema mirrors the CreatePollPayload the Go handler accepts.
// Deadline is split into date (yyyy-MM-dd) + time (HH:mm) fields backing the
// custom pickers; the dialog composes them into an ISO string at submit.
export const createPollSchema = z
  .object({
    type: pollTypeEnum,
    title: z.string().trim().min(1, "Required").max(200, "Too long"),
    description: z.string().trim().max(2000, "Too long"),
    is_anonymous: z.boolean(),
    max_choices: z
      .string()
      .trim()
      .min(1, "Required")
      .refine((v) => {
        const n = Number(v);
        return Number.isInteger(n) && n >= 1 && n <= 20;
      }, "Between 1 and 20"),
    allow_option_add: z.boolean(),
    result_visibility: pollVisibilityEnum,
    deadline_date: z.string(),
    deadline_time: z.string(),
    options: z
      .array(
        z.object({
          text: z.string().trim().min(1, "Required").max(200, "Too long"),
        }),
      )
      .max(50, "Too many options"),
  })
  .refine((d) => d.options.length >= 2 || d.allow_option_add, {
    message: "Add at least two options or allow members to add later",
    path: ["options"],
  })
  .refine(
    (d) => (d.deadline_date === "") === (d.deadline_time === ""),
    { message: "Pick both a date and a time", path: ["deadline_time"] },
  )
  .refine(
    (d) => {
      if (!d.deadline_date || !d.deadline_time) return true;
      const t = new Date(`${d.deadline_date}T${d.deadline_time}`).getTime();
      return Number.isFinite(t) && t > Date.now();
    },
    { message: "Deadline must be in the future", path: ["deadline_time"] },
  );

export type CreatePollFormInput = z.infer<typeof createPollSchema>;

export const addOptionSchema = z.object({
  text: z.string().trim().min(1, "Required").max(200, "Too long"),
});
export type AddOptionFormInput = z.infer<typeof addOptionSchema>;
