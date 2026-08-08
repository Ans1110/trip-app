"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Loader2, ShieldAlert, Lock } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import {
  elevateSchema,
  type ElevateInput,
} from "@/lib/schemas/admin-schemas";
import { adminApi } from "@/lib/apis/admin-api";
import { adminKeys, errorMessage, useElevate } from "@/hooks/admin-hooks";
import { useMe } from "@/hooks/auth-hooks";
import { ApiError } from "@/lib/utils";

const probeKey = [...adminKeys.all, "probe"] as const;

export function AdminGate({ children }: { children: React.ReactNode }) {
  const { data: user, isLoading: meLoading } = useMe();
  const isAdmin = !!user?.is_admin;

  if (meLoading) return <Checking />;
  if (!isAdmin) return <DeniedState />;
  return <AdminProbe>{children}</AdminProbe>;
}

function AdminProbe({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const probe = useQuery<boolean, ApiError>({
    queryKey: probeKey,
    queryFn: async ({ signal }) => {
      await adminApi.listUsers({ limit: 1 }, signal);
      return true;
    },
    retry: false,
    staleTime: Infinity,
  });

  if (probe.isLoading) return <Checking />;
  if (probe.isSuccess) return <>{children}</>;

  const err = probe.error;
  if (err instanceof ApiError && err.status === 401) return <DeniedState />;

  return (
    <ElevateCard
      onElevated={() => qc.invalidateQueries({ queryKey: probeKey })}
      probeError={
        err instanceof ApiError && err.status === 403
          ? null
          : errorMessage(err)
      }
    />
  );
}

function Checking() {
  return (
    <div
      className="flex items-center gap-2 text-sm px-6 py-12"
      style={{ color: "#8B9A8E" }}
    >
      <Loader2 className="size-4 animate-spin" />
      Checking admin access
    </div>
  );
}

function DeniedState() {
  return (
    <div className="max-w-md mx-auto mt-16 px-6">
      <div
        className="flex flex-col items-center text-center gap-3 p-8 rounded-2xl border"
        style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
      >
        <ShieldAlert className="size-10" style={{ color: "#FCA5A5" }} />
        <h2 className="text-lg font-semibold" style={{ color: "#ECEFEA" }}>
          Not authorized
        </h2>
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Your account can&rsquo;t access the admin console.
        </p>
      </div>
    </div>
  );
}

function ElevateCard({
  onElevated,
  probeError,
}: {
  onElevated: () => void;
  probeError: string | null;
}) {
  const form = useForm<ElevateInput>({
    resolver: zodResolver(elevateSchema),
    defaultValues: { password: "" },
  });
  const mutation = useElevate({ onSuccess: () => onElevated() });
  const submitError = errorMessage(mutation.error) ?? probeError;

  return (
    <div className="max-w-md mx-auto mt-16 px-6">
      <div
        className="p-8 rounded-2xl border"
        style={{ backgroundColor: "#121814", borderColor: "#1F2A24" }}
      >
        <div className="flex items-center gap-3 mb-6">
          <Lock className="size-5" style={{ color: "var(--season-button)" }} />
          <div>
            <h2
              className="text-lg font-semibold"
              style={{ color: "#ECEFEA" }}
            >
              Admin sign-in
            </h2>
            <p className="text-xs" style={{ color: "#8B9A8E" }}>
              Enter the admin password to unlock this session.
            </p>
          </div>
        </div>
        <FormProvider {...form}>
          <form
            onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
            className="flex flex-col gap-5"
            noValidate
          >
            <ControlledInput<ElevateInput>
              name="password"
              label="Password"
              type="password"
              autoComplete="current-password"
              autoFocus
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
                  Verifying
                </>
              ) : (
                <>
                  Unlock <ArrowRight className="size-4" />
                </>
              )}
            </button>
          </form>
        </FormProvider>
      </div>
    </div>
  );
}
