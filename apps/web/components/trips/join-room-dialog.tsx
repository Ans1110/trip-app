"use client";

import { useRouter } from "next/navigation";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { errorMessage, useJoinRoom } from "@/hooks/trip-hooks";
import { joinRoomSchema, type JoinRoomFormInput } from "@/lib/trip-schemas";

import { Modal } from "./modal";

export function JoinRoomDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const join = useJoinRoom();
  const form = useForm<JoinRoomFormInput>({
    resolver: zodResolver(joinRoomSchema),
    defaultValues: { code: "" },
  });
  const code = form.watch("code");

  const onSubmit = (v: JoinRoomFormInput) => {
    join.mutate(
      { code: v.code },
      {
        onSuccess: (result) => {
          onClose();
          router.push(`/trips/${result.trip_id}`);
        },
      },
    );
  };

  const submitError = errorMessage(join.error);

  return (
    <Modal title="Join a trip" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <p className="text-sm text-[#8B9A8E]">
            Enter the room code shared with you, or scan a friend&rsquo;s QR
            code.
          </p>

          <ControlledInput<JoinRoomFormInput>
            name="code"
            label="Room code"
            placeholder="ABCD1234"
            maxLength={16}
            autoFocus
            className="uppercase tracking-widest"
          />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={join.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={join.isPending || !code.trim()}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {join.isPending && <Loader2 className="size-3.5 animate-spin" />}
              Join
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}
