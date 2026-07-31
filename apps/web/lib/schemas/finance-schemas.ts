import { z } from "zod";

// Decimal fields are strings on the wire; the schema below enforces a
// well-formed positive decimal so the FE never sends "12,50" or "abc" through
// to the Go handler.
const decimalString = (opts?: { maxFractionDigits?: number }) => {
  const max = opts?.maxFractionDigits ?? 2;
  const re = new RegExp(`^\\d+(\\.\\d{1,${max}})?$`);
  return z
    .string()
    .trim()
    .min(1, "Required")
    .refine((v) => re.test(v), "Enter a number like 12.50")
    .refine((v) => Number(v) > 0, "Must be greater than 0");
};

export const createExpenseSchema = z.object({
  amount: decimalString(),
  description: z.string().trim().max(200, "Too long"),
  category: z.string().trim().max(60, "Too long"),
  occurred_date: z.string().min(1, "Pick a date"),
  occurred_time: z.string().min(1, "Pick a time"),
  participants: z.array(z.string().uuid()).min(1, "Pick at least one person"),
});

export type CreateExpenseFormInput = z.infer<typeof createExpenseSchema>;

export const upsertBudgetSchema = z.object({
  category: z.string().trim().min(1, "Required").max(60, "Too long"),
  amount: decimalString(),
});
export type UpsertBudgetFormInput = z.infer<typeof upsertBudgetSchema>;
