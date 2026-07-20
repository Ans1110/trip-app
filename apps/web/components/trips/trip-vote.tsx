"use client";

import { useMemo, useState } from "react";
import { FormProvider, useFieldArray, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, Plus, Trash2, Vote } from "lucide-react";

import { ControlledCheckbox } from "@/components/ui/controlled-checkbox";
import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { ControlledTimePicker } from "@/components/ui/controlled-time-picker";
import { useMe } from "@/hooks/auth-hooks";
import { useTripRealtime } from "@/hooks/realtime-hooks";
import { useRoom } from "@/hooks/trip-hooks";
import { errorMessage, useCreatePoll, usePolls } from "@/hooks/vote-hooks";
import type { UserSummary } from "@/lib/apis/trip-api";
import type {
  CreatePollInput,
  PollResultVisibility,
  PollType,
} from "@/lib/apis/vote-api";
import {
  createPollSchema,
  type CreatePollFormInput,
} from "@/lib/schemas/vote-schemas";

import { Modal } from "./modal";
import { PollCard } from "./poll-card";

const POLL_TYPES: { value: PollType; label: string }[] = [
  { value: "custom", label: "Custom" },
  { value: "location", label: "Location" },
  { value: "time", label: "Time" },
];

const VISIBILITY_OPTIONS: { value: PollResultVisibility; label: string }[] = [
  { value: "always", label: "Show results while open" },
  { value: "after_vote", label: "Show after voting" },
  { value: "after_deadline", label: "Show after close" },
];

export function TripVote({ tripId }: { tripId: string }) {
  useTripRealtime(tripId);

  const pollsQuery = usePolls(tripId);
  const meQuery = useMe();
  const roomQuery = useRoom(tripId);
  const [creating, setCreating] = useState(false);

  const memberById = useMemo(() => {
    const map = new Map<string, UserSummary>();
    for (const m of roomQuery.data?.members ?? []) {
      map.set(m.user.id, m.user);
    }
    return map;
  }, [roomQuery.data]);

  const polls = pollsQuery.data ?? [];
  const openPolls = polls.filter((p) => p.status === "open");
  const closedPolls = polls.filter((p) => p.status === "closed");

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {polls.length === 0
            ? "No polls yet — start one to gather the group's picks."
            : `${openPolls.length} open · ${closedPolls.length} closed`}
        </p>
        <button
          type="button"
          onClick={() => setCreating(true)}
          className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full text-[#0B100D]"
          style={{ backgroundColor: "var(--season-button)" }}
        >
          <Plus className="size-4" />
          New poll
        </button>
      </div>

      {pollsQuery.isLoading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {pollsQuery.isError && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(pollsQuery.error)}
        </p>
      )}

      {!pollsQuery.isLoading && !pollsQuery.isError && polls.length === 0 && (
        <EmptyState onCreate={() => setCreating(true)} />
      )}

      {openPolls.length > 0 && (
        <div className="flex flex-col gap-3">
          {openPolls.map((p) => (
            <PollCard
              key={p.id}
              tripId={tripId}
              poll={p}
              meId={meQuery.data?.id}
              memberById={memberById}
            />
          ))}
        </div>
      )}

      {closedPolls.length > 0 && (
        <div className="flex flex-col gap-3">
          <h3 className="text-xs uppercase tracking-wider" style={{ color: "#6B7A6F" }}>
            Closed
          </h3>
          {closedPolls.map((p) => (
            <PollCard
              key={p.id}
              tripId={tripId}
              poll={p}
              meId={meQuery.data?.id}
              memberById={memberById}
            />
          ))}
        </div>
      )}

      {creating && (
        <CreatePollDialog tripId={tripId} onClose={() => setCreating(false)} />
      )}
    </section>
  );
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div
      className="flex flex-col items-center gap-3 py-10 rounded-xl"
      style={{ backgroundColor: "#121814", border: "1px dashed #1F2A24" }}
    >
      <div
        className="size-12 rounded-full flex items-center justify-center"
        style={{
          backgroundColor:
            "color-mix(in srgb, var(--season-button) 18%, transparent)",
        }}
      >
        <Vote className="size-5" style={{ color: "#ECEFEA" }} />
      </div>
      <p className="text-sm" style={{ color: "#ECEFEA" }}>
        Nothing to vote on yet
      </p>
      <button
        type="button"
        onClick={onCreate}
        className="text-xs px-3 py-1.5 rounded-full hover:bg-white/5"
        style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
      >
        Create the first poll
      </button>
    </div>
  );
}

function CreatePollDialog({
  tripId,
  onClose,
}: {
  tripId: string;
  onClose: () => void;
}) {
  const create = useCreatePoll();

  const form = useForm<CreatePollFormInput>({
    resolver: zodResolver(createPollSchema),
    defaultValues: {
      type: "custom",
      title: "",
      description: "",
      is_anonymous: false,
      max_choices: "1",
      allow_option_add: false,
      result_visibility: "always",
      deadline_date: "",
      deadline_time: "",
      options: [{ text: "" }, { text: "" }],
    },
  });
  const {
    handleSubmit,
    control,
    formState: { errors },
  } = form;

  const { fields, append, remove } = useFieldArray({
    control,
    name: "options",
  });

  const onSubmit = (v: CreatePollFormInput) => {
    const trimmedOptions = v.options
      .map((o) => ({ text: o.text.trim() }))
      .filter((o) => o.text.length > 0);

    const input: CreatePollInput = {
      type: v.type,
      title: v.title.trim(),
      description: v.description.trim() || undefined,
      is_anonymous: v.is_anonymous,
      max_choices: Number(v.max_choices),
      allow_option_add: v.allow_option_add,
      result_visibility: v.result_visibility,
      deadline_at:
        v.deadline_date && v.deadline_time
          ? new Date(`${v.deadline_date}T${v.deadline_time}`).toISOString()
          : undefined,
      options: trimmedOptions.length > 0 ? trimmedOptions : undefined,
    };

    create.mutate(
      { tripId, input },
      { onSuccess: () => onClose() },
    );
  };

  const submitError = errorMessage(create.error);

  return (
    <Modal title="New poll" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<CreatePollFormInput>
            name="title"
            label="Question"
            placeholder="Where should we stay in Tokyo?"
          />

          <ControlledTextarea<CreatePollFormInput>
            name="description"
            label="Details (optional)"
            rows={2}
            placeholder="Extra context for voters"
          />

          <div className="grid grid-cols-2 gap-3">
            <ControlledSelect<CreatePollFormInput>
              name="type"
              label="Kind"
              options={POLL_TYPES}
            />
            <ControlledInput<CreatePollFormInput>
              name="max_choices"
              label="Max picks per voter"
              inputMode="numeric"
              placeholder="1"
            />
          </div>

          <ControlledSelect<CreatePollFormInput>
            name="result_visibility"
            label="Result visibility"
            options={VISIBILITY_OPTIONS}
          />

          <div className="flex flex-col gap-1.5">
            <span className="text-sm text-[#ECEFEA]">
              Deadline{" "}
              <span className="text-xs" style={{ color: "#6B7A6F" }}>
                (optional)
              </span>
            </span>
            <div className="grid grid-cols-2 gap-3">
              <ControlledDatePicker<CreatePollFormInput>
                name="deadline_date"
                placeholder="Pick a date"
              />
              <ControlledTimePicker<CreatePollFormInput>
                name="deadline_time"
                placeholder="Pick a time"
              />
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <ControlledCheckbox<CreatePollFormInput>
              name="is_anonymous"
              label="Hide who voted for what"
            />
            <ControlledCheckbox<CreatePollFormInput>
              name="allow_option_add"
              label="Let members add options later"
            />
          </div>

          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-[#ECEFEA]">Options</span>
              <button
                type="button"
                onClick={() => append({ text: "" })}
                className="text-xs inline-flex items-center gap-1 px-2 py-1 rounded-full hover:bg-white/5"
                style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
              >
                <Plus className="size-3" />
                Add option
              </button>
            </div>
            <div className="flex flex-col gap-2">
              {fields.map((field, i) => (
                <div key={field.id} className="flex items-start gap-2">
                  <ControlledInput<CreatePollFormInput>
                    name={`options.${i}.text` as const}
                    placeholder={`Option ${i + 1}`}
                    containerClassName="flex-1"
                  />
                  <button
                    type="button"
                    onClick={() => remove(i)}
                    disabled={fields.length <= 1}
                    aria-label="Remove option"
                    className="inline-flex items-center justify-center size-9 rounded-full hover:bg-white/5 disabled:opacity-40"
                    style={{ color: "#FCA5A5" }}
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                </div>
              ))}
            </div>
            {errors.options?.message && (
              <p className="text-[11px] text-[#FCA5A5]">
                {errors.options.message}
              </p>
            )}
          </div>

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={create.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={create.isPending}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {create.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              Create poll
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}
