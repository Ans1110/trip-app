import { z } from "zod";

export const createPackingItemSchema = z.object({
  name: z.string().trim().min(1, "Required").max(200, "Too long"),
  quantity: z
    .string()
    .trim()
    .refine((v) => v === "" || Number.isInteger(Number(v)), "Must be a number")
    .refine(
      (v) => v === "" || (Number(v) >= 1 && Number(v) <= 999),
      "Between 1 and 999",
    ),
  category: z.string().trim().max(32, "Too long"),
  note: z.string().trim().max(2000, "Too long"),
});
export type CreatePackingItemFormInput = z.infer<
  typeof createPackingItemSchema
>;

export const updatePackingItemSchema = z.object({
  name: z.string().trim().min(1, "Required").max(200, "Too long"),
  quantity: z
    .string()
    .trim()
    .refine((v) => Number.isInteger(Number(v)), "Must be a number")
    .refine((v) => Number(v) >= 1 && Number(v) <= 999, "Between 1 and 999"),
  category: z.string().trim().max(32, "Too long"),
  note: z.string().trim().max(2000, "Too long"),
});
export type UpdatePackingItemFormInput = z.infer<
  typeof updatePackingItemSchema
>;
