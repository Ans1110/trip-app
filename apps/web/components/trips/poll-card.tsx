"use client";

import { useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  CheckCircle2,
  Circle,
  Clock,
  EyeOff,
  Loader2,
  Lock,
  Plus,
  Square,
  SquareCheckBig,
  Trash2,
  Users,
  X,
} from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import {
  errorMessage,
  useAddOption,
  useCastVote,
  useClosePoll,
  useDeleteOption,
  useDeletePoll,
} from "@/hooks/vote-hooks";
import type { Poll, PollOption } from "@/lib/apis/vote-api";
import type { UserSummary } from "@/lib/apis/trip-api";
import {
  addOptionSchema,
  type AddOptionFormInput,
} from "@/lib/schemas/vote-schemas";

type MemberLookup = Map<string, UserSummary>;

const typeBadge: Record<Poll["type"], string> = {
  location: "Location",
  time: "Time",
  custom: "Custom",
};

const visibilityLabel: Record<Poll["result_visibility"], string> = {
  always: "Live results",
  after_vote: "Results after your vote",
  after_deadline: "Results after close",
};

export function PollCard({
  tripId,
  poll,
  meId,
  memberById,
}: {
  tripId: string;
  poll: Poll;
  meId: string | undefined;
  memberById: MemberLookup;
}) {
  const isCreator = meId != null && meId === poll.created_by;
  const isMulti = poll.max_choices > 1;
  const isClosed = poll.status === "closed";
  const deadlinePassed =
    poll.deadline_at != null && new Date(poll.deadline_at).getTime() <= Date.now();
  const canVote = !isClosed && !deadlinePassed;
  const canAddOption = canVote && (isCreator || poll.allow_option_add);

  const totalVotes = useMemo(
    () => poll.options.reduce((s, o) => s + (o.count ?? 0), 0),
    [poll.options],
  );

  const cast = useCastVote();
  const closeMut = useClosePoll();
  const removeMut = useDeletePoll();
  const delOption = useDeleteOption();

  const [draft, setDraft] = useState<Set<string>>(
    () => new Set(poll.my_choices),
  );
  const hydrated = useMemo(() => new Set(poll.my_choices), [poll.my_choices]);
  const draftChanged = useMemo(() => {
    if (draft.size !== hydrated.size) return true;
    for (const id of draft) if (!hydrated.has(id)) return true;
    return false;
  }, [draft, hydrated]);

  const submitVote = (ids: string[]) => {
    if (!canVote) return;
    cast.mutate({ tripId, pollId: poll.id, input: { option_ids: ids } });
  };

  const handlePick = (optId: string) => {
    if (!canVote) return;
    if (isMulti) {
      setDraft((prev) => {
        const next = new Set(prev);
        if (next.has(optId)) next.delete(optId);
        else if (next.size < poll.max_choices) next.add(optId);
        return next;
      });
    } else {
      setDraft(new Set([optId]));
      submitVote([optId]);
    }
  };

  const handleRemoveOption = (optId: string) => {
    if (!window.confirm("Remove this option?")) return;
    delOption.mutate({ tripId, pollId: poll.id, optionId: optId });
  };

  const handleClose = () => {
    if (!window.confirm("Close this poll? Voting will stop immediately.")) return;
    closeMut.mutate({ tripId, pollId: poll.id });
  };
  const handleDelete = () => {
    if (!window.confirm("Delete this poll and all its votes?")) return;
    removeMut.mutate({ tripId, pollId: poll.id });
  };

  return (
    <article
      className="rounded-xl p-4 flex flex-col gap-3"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <header className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="text-sm font-medium text-[#ECEFEA] truncate">
              {poll.title}
            </h3>
            <Badge>{typeBadge[poll.type]}</Badge>
            {poll.is_anonymous && (
              <Badge muted>
                <EyeOff className="size-3" /> Anonymous
              </Badge>
            )}
            {isMulti && <Badge muted>Pick up to {poll.max_choices}</Badge>}
            {isClosed && (
              <Badge tone="closed">
                <Lock className="size-3" /> Closed
              </Badge>
            )}
          </div>
          {poll.description && (
            <p className="text-xs text-[#8B9A8E]">{poll.description}</p>
          )}
          <DeadlineLine deadline={poll.deadline_at} closed={isClosed} />
        </div>
        {isCreator && (
          <div className="flex items-center gap-1 shrink-0">
            {!isClosed && (
              <IconButton
                aria-label="Close poll"
                onClick={handleClose}
                busy={closeMut.isPending}
                color="#B5D086"
              >
                <Lock className="size-3.5" />
              </IconButton>
            )}
            <IconButton
              aria-label="Delete poll"
              onClick={handleDelete}
              busy={removeMut.isPending}
              color="#FCA5A5"
            >
              <Trash2 className="size-3.5" />
            </IconButton>
          </div>
        )}
      </header>

      <ul className="flex flex-col gap-1.5">
        {poll.options.map((opt) => (
          <OptionRow
            key={opt.id}
            poll={poll}
            option={opt}
            picked={draft.has(opt.id)}
            isMulti={isMulti}
            canVote={canVote}
            totalVotes={totalVotes}
            memberById={memberById}
            meId={meId}
            onPick={() => handlePick(opt.id)}
            onDelete={
              (isCreator || opt.added_by === meId) && !isClosed
                ? () => handleRemoveOption(opt.id)
                : undefined
            }
          />
        ))}
      </ul>

      {isMulti && canVote && (
        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={cast.isPending || !draftChanged || draft.size === 0}
            onClick={() => submitVote([...draft])}
            className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full disabled:opacity-50 text-[#0B100D]"
            style={{ backgroundColor: "var(--season-button)" }}
          >
            {cast.isPending ? (
              <Loader2 className="size-3 animate-spin" />
            ) : (
              <CheckCircle2 className="size-3" />
            )}
            Save vote
          </button>
          {draftChanged && (
            <button
              type="button"
              onClick={() => setDraft(new Set(poll.my_choices))}
              className="text-xs text-[#8B9A8E] hover:text-[#ECEFEA]"
            >
              Reset
            </button>
          )}
        </div>
      )}

      {canAddOption && <AddOptionForm tripId={tripId} pollId={poll.id} />}

      <footer className="flex items-center justify-between text-[11px] text-[#6B7A6F]">
        <span className="inline-flex items-center gap-1">
          <Users className="size-3" />
          {poll.total_voters} voter{poll.total_voters === 1 ? "" : "s"}
        </span>
        <span>{visibilityLabel[poll.result_visibility]}</span>
      </footer>

      {cast.isError && (
        <p className="text-[11px] text-[#FCA5A5]">{errorMessage(cast.error)}</p>
      )}
      {delOption.isError && (
        <p className="text-[11px] text-[#FCA5A5]">
          {errorMessage(delOption.error)}
        </p>
      )}
    </article>
  );
}

function OptionRow({
  poll,
  option,
  picked,
  isMulti,
  canVote,
  totalVotes,
  memberById,
  meId,
  onPick,
  onDelete,
}: {
  poll: Poll;
  option: PollOption;
  picked: boolean;
  isMulti: boolean;
  canVote: boolean;
  totalVotes: number;
  memberById: MemberLookup;
  meId: string | undefined;
  onPick: () => void;
  onDelete?: () => void;
}) {
  const percent =
    poll.results_visible && totalVotes > 0
      ? Math.round((option.count * 100) / totalVotes)
      : null;

  return (
    <li>
      <div className="relative overflow-hidden rounded-lg">
        {percent != null && (
          <div
            aria-hidden
            className="absolute inset-y-0 left-0 season-transition"
            style={{
              width: `${percent}%`,
              backgroundColor: picked
                ? "color-mix(in srgb, var(--season-button) 25%, transparent)"
                : "rgba(139,154,142,0.12)",
            }}
          />
        )}
        <div
          className="relative flex items-center gap-2.5 px-3 py-2"
          style={{
            border: `1px solid ${picked ? "var(--season-button)" : "#1F2A24"}`,
            borderRadius: 8,
          }}
        >
          <button
            type="button"
            onClick={onPick}
            disabled={!canVote}
            aria-pressed={picked}
            className="flex items-center gap-2 flex-1 min-w-0 text-left disabled:cursor-not-allowed"
          >
            <PickIcon picked={picked} isMulti={isMulti} />
            <span className="text-sm text-[#ECEFEA] truncate">
              {option.text}
            </span>
          </button>
          {poll.results_visible && (
            <span className="text-xs tabular-nums text-[#8B9A8E]">
              {option.count}
              {percent != null ? ` · ${percent}%` : ""}
            </span>
          )}
          {onDelete && (
            <button
              type="button"
              onClick={onDelete}
              aria-label="Remove option"
              className="inline-flex items-center justify-center size-6 rounded-full hover:bg-white/5"
              style={{ color: "#FCA5A5" }}
            >
              <X className="size-3" />
            </button>
          )}
        </div>
      </div>
      {poll.results_visible &&
        !poll.is_anonymous &&
        option.voters &&
        option.voters.length > 0 && (
          <div className="flex flex-wrap gap-1 pl-8 pt-1">
            {option.voters.map((uid) => {
              const name = memberById.get(uid)?.name ?? "member";
              const mine = uid === meId;
              return (
                <span
                  key={uid}
                  className="text-[10px] px-1.5 py-0.5 rounded-full"
                  style={{
                    backgroundColor: mine
                      ? "color-mix(in srgb, var(--season-button) 20%, transparent)"
                      : "#1F2A24",
                    color: mine ? "#ECEFEA" : "#9CB0A3",
                  }}
                >
                  {mine ? "You" : name}
                </span>
              );
            })}
          </div>
        )}
    </li>
  );
}

function PickIcon({ picked, isMulti }: { picked: boolean; isMulti: boolean }) {
  if (isMulti) {
    return picked ? (
      <SquareCheckBig
        className="size-4"
        style={{ color: "var(--season-button)" }}
      />
    ) : (
      <Square className="size-4 text-[#6B7A6F]" />
    );
  }
  return picked ? (
    <CheckCircle2
      className="size-4"
      style={{ color: "var(--season-button)" }}
    />
  ) : (
    <Circle className="size-4 text-[#6B7A6F]" />
  );
}

function AddOptionForm({ tripId, pollId }: { tripId: string; pollId: string }) {
  const add = useAddOption();
  const form = useForm<AddOptionFormInput>({
    resolver: zodResolver(addOptionSchema),
    defaultValues: { text: "" },
  });
  const {
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = form;
  const text = watch("text");

  const onSubmit = (v: AddOptionFormInput) => {
    add.mutate(
      { tripId, pollId, input: { text: v.text } },
      { onSuccess: () => reset({ text: "" }) },
    );
  };

  return (
    <FormProvider {...form}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="flex items-start gap-2"
      >
        <ControlledInput<AddOptionFormInput>
          name="text"
          placeholder="Add another option…"
          containerClassName="flex-1"
        />
        <button
          type="submit"
          disabled={add.isPending || !text.trim()}
          className="season-transition inline-flex items-center gap-1 px-3 py-2 text-xs font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
          style={{ backgroundColor: "var(--season-button)" }}
        >
          {add.isPending ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <Plus className="size-3.5" />
          )}
          Add
        </button>
        {errors.text && (
          <span className="text-[11px] text-[#FCA5A5]">
            {errors.text.message}
          </span>
        )}
        {add.isError && (
          <span className="text-[11px] text-[#FCA5A5]">
            {errorMessage(add.error)}
          </span>
        )}
      </form>
    </FormProvider>
  );
}

function DeadlineLine({
  deadline,
  closed,
}: {
  deadline: string | undefined;
  closed: boolean;
}) {
  if (!deadline) return null;
  const t = new Date(deadline).getTime();
  const now = Date.now();
  const past = t <= now;
  const label = past
    ? closed
      ? `Closed · ${formatShort(deadline)}`
      : "Deadline passed · closing soon"
    : `Closes ${formatDistance(t - now)}`;
  return (
    <span
      className="inline-flex items-center gap-1 text-[11px]"
      style={{ color: past ? "#FCA5A5" : "#8B9A8E" }}
    >
      <Clock className="size-3" />
      {label}
    </span>
  );
}

function IconButton({
  children,
  onClick,
  busy,
  color,
  "aria-label": ariaLabel,
}: {
  children: React.ReactNode;
  onClick: () => void;
  busy?: boolean;
  color: string;
  "aria-label": string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      disabled={busy}
      className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60"
      style={{ color }}
    >
      {busy ? <Loader2 className="size-3.5 animate-spin" /> : children}
    </button>
  );
}

function Badge({
  children,
  muted,
  tone,
}: {
  children: React.ReactNode;
  muted?: boolean;
  tone?: "closed";
}) {
  const style =
    tone === "closed"
      ? { bg: "rgba(252,165,165,0.14)", fg: "#FCA5A5" }
      : muted
        ? { bg: "#1F2A24", fg: "#9CB0A3" }
        : {
            bg: "color-mix(in srgb, var(--season-button) 18%, transparent)",
            fg: "#ECEFEA",
          };
  return (
    <span
      className="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full"
      style={{ backgroundColor: style.bg, color: style.fg }}
    >
      {children}
    </span>
  );
}

function formatDistance(ms: number): string {
  const abs = Math.abs(ms);
  const minute = 60_000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (abs < hour) return `in ${Math.max(1, Math.round(abs / minute))}m`;
  if (abs < day) return `in ${Math.round(abs / hour)}h`;
  return `in ${Math.round(abs / day)}d`;
}

function formatShort(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}
