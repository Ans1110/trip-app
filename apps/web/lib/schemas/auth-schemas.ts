import { z } from "zod";

const email = z.string().trim().email("Enter a valid email");
const strongPassword = z
  .string()
  .min(8, "At least 8 characters")
  .max(72, "Too long");

export const mfaCodeSchema = z
  .string()
  .trim()
  .regex(/^\d{6}$/, "Enter the 6-digit code");

export const signInSchema = z.object({
  email,
  password: z.string().min(1, "Required"),
  mfa_code: z.string().optional(),
});
export type SignInInput = z.infer<typeof signInSchema>;

export const mfaChallengeSchema = z.object({
  mfa_code: mfaCodeSchema,
});
export type MFAChallengeInput = z.infer<typeof mfaChallengeSchema>;

export const signUpSchema = z.object({
  name: z.string().trim().min(1, "Required").max(50, "Too long"),
  email,
  password: strongPassword,
});
export type SignUpInput = z.infer<typeof signUpSchema>;

export const forgotPasswordSchema = z.object({ email });
export type ForgotPasswordInput = z.infer<typeof forgotPasswordSchema>;

export const resetPasswordSchema = z
  .object({
    password: strongPassword,
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, {
    message: "Passwords don't match",
    path: ["confirm"],
  });
export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>;

export const resendVerificationSchema = z.object({ email });
export type ResendVerificationInput = z.infer<typeof resendVerificationSchema>;
