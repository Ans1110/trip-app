"use client";

import Link from "next/link";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Check, Loader2 } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import {
  resetPasswordSchema,
  type ResetPasswordInput,
} from "@/lib/auth-schemas";
import { errorMessage, useResetPassword } from "@/hooks/auth-hooks";

const AUTH_LABEL = "text-[11px] tracking-[0.2em] uppercase";
const AUTH_INPUT = "px-4 py-3";

export function ResetPasswordForm({ token }: { token: string }) {
  const form = useForm<ResetPasswordInput>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: { password: "", confirm: "" },
  });

  const mutation = useResetPassword();

  if (mutation.isSuccess) {
    return (
      <div
        className="season-transition flex flex-col gap-4 rounded-2xl px-5 py-6"
        style={{
          backgroundColor:
            "color-mix(in srgb, var(--season-accent) 8%, transparent)",
          border:
            "1px solid color-mix(in srgb, var(--season-accent) 28%, transparent)",
        }}
        role="status"
      >
        <div className="flex items-center gap-2">
          <Check
            className="season-transition size-4"
            style={{ color: "var(--season-accent)" }}
          />
          <p
            className="season-transition text-[11px] tracking-[0.2em] uppercase font-medium"
            style={{ color: "var(--season-accent)" }}
          >
            Password updated
          </p>
        </div>
        <p className="text-sm" style={{ color: "#ECEFEA" }}>
          You can sign in with your new password now.
        </p>
        <Link
          href="/sign-in"
          className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
            boxShadow: "0 8px 24px var(--season-button-shadow)",
          }}
        >
          Sign in <ArrowRight className="size-4" />
        </Link>
      </div>
    );
  }

  const submitError = errorMessage(mutation.error);

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit((v) =>
          mutation.mutate({ token, password: v.password }),
        )}
        className="flex flex-col gap-5"
        noValidate
      >
        <ControlledInput<ResetPasswordInput>
          name="password"
          label="New password"
          type="password"
          autoComplete="new-password"
          hint="At least 8 characters."
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
        />
        <ControlledInput<ResetPasswordInput>
          name="confirm"
          label="Confirm password"
          type="password"
          autoComplete="new-password"
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
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
          className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
            boxShadow: "0 8px 24px var(--season-button-shadow)",
          }}
        >
          {mutation.isPending ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Updating
            </>
          ) : (
            <>
              Update password <ArrowRight className="size-4" />
            </>
          )}
        </button>
      </form>
    </FormProvider>
  );
}
