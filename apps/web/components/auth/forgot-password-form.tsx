"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Check, Loader2 } from "lucide-react";

import {
  forgotPasswordSchema,
  type ForgotPasswordInput,
} from "@/lib/auth-schemas";
import { errorMessage, useForgotPassword } from "@/hooks/auth-hooks";

export function ForgotPasswordForm() {
  const {
    register,
    handleSubmit,
    getValues,
    reset,
    formState: { errors },
  } = useForm<ForgotPasswordInput>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { email: "" },
  });

  const mutation = useForgotPassword();

  if (mutation.isSuccess) {
    const email = getValues("email");
    return (
      <div
        className="flex flex-col gap-3 rounded-2xl px-5 py-6"
        style={{
          backgroundColor: "rgba(127,182,138,0.06)",
          border: "1px solid rgba(127,182,138,0.25)",
        }}
        role="status"
      >
        <div className="flex items-center gap-2">
          <Check className="size-4" style={{ color: "#A8E0B4" }} />
          <p
            className="text-[11px] tracking-[0.2em] uppercase font-medium"
            style={{ color: "#A8E0B4" }}
          >
            Check your inbox
          </p>
        </div>
        <p className="text-sm" style={{ color: "#ECEFEA" }}>
          If an account exists for <strong>{email}</strong>, we&apos;ve sent a
          link to reset its password. It expires in 30 minutes.
        </p>
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          Didn&apos;t get it? Check spam, or{" "}
          <button
            type="button"
            onClick={() => {
              mutation.reset();
              reset({ email: "" });
            }}
            className="underline-offset-4 hover:underline"
            style={{ color: "#A8E0B4" }}
          >
            try again
          </button>
          .
        </p>
      </div>
    );
  }

  const submitError = errorMessage(mutation.error);

  return (
    <form
      onSubmit={handleSubmit((v) => mutation.mutate(v))}
      className="flex flex-col gap-5"
      noValidate
    >
      <Field
        id="email"
        label="Email"
        type="email"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
      />

      {submitError && (
        <p
          className="text-sm rounded-lg px-3 py-2"
          style={{
            color: "#FCA5A5",
            backgroundColor: "rgba(220,38,38,0.08)",
            border: "1px solid rgba(220,38,38,0.25)",
          }}
          role="alert"
        >
          {submitError}
        </p>
      )}

      <button
        type="submit"
        disabled={mutation.isPending}
        className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide transition-all hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
        style={{
          backgroundColor: "#7FB68A",
          color: "#0B100D",
          boxShadow: "0 8px 24px rgba(127,182,138,0.18)",
        }}
      >
        {mutation.isPending ? (
          <>
            <Loader2 className="size-4 animate-spin" />
            Sending link
          </>
        ) : (
          <>
            Send reset link <ArrowRight className="size-4" />
          </>
        )}
      </button>
    </form>
  );
}

type FieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  id: string;
  label: string;
  error?: string;
};

function Field({ id, label, error, ...input }: FieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor={id}
        className="text-[11px] tracking-[0.2em] uppercase font-medium"
        style={{ color: "#8B9A8E" }}
      >
        {label}
      </label>
      <input
        id={id}
        {...input}
        className="px-4 py-3 rounded-lg text-sm outline-none transition-colors focus:border-[#7FB68A]"
        style={{
          backgroundColor: "#161E19",
          border: `1px solid ${error ? "rgba(220,38,38,0.4)" : "#1F2A24"}`,
          color: "#ECEFEA",
        }}
        aria-invalid={error ? true : undefined}
      />
      {error && (
        <p className="text-[11px]" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      )}
    </div>
  );
}
