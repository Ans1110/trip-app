"use client";

import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { DropdownMenu as RxMenu } from "radix-ui";
import {
  Ban,
  Check,
  Flag,
  Loader2,
  MoreHorizontal,
  Share2,
  UserPlus,
  UserMinus,
} from "lucide-react";

import { Modal } from "@/components/trips/modal";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { useBlockUser } from "@/hooks/friend-hooks";
import {
  useFollow,
  useProfileByUsername,
  useUnfollow,
} from "@/hooks/profile-hooks";
import { errorMessage, useSubmitReport } from "@/hooks/admin-hooks";
import {
  createReportSchema,
  type CreateReportInput,
} from "@/lib/schemas/admin-schemas";

type Props = {
  postId: string;
  authorUserId: string;
  authorUsername?: string;
  authorName: string;
};

export function PostActionsMenu({
  postId,
  authorUserId,
  authorUsername,
  authorName,
}: Props) {
  const [open, setOpen] = useState(false);
  const [blockOpen, setBlockOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const [reported, setReported] = useState(false);
  const [shareState, setShareState] = useState<"idle" | "copied" | "failed">(
    "idle",
  );

  const profile = useProfileByUsername(authorUsername ?? "", {
    enabled: open && !!authorUsername,
  });
  const follow = useFollow();
  const unfollow = useUnfollow();
  const block = useBlockUser();

  const isFollowing = profile.data?.is_following ?? false;
  const followBusy =
    profile.isLoading || follow.isPending || unfollow.isPending;

  const onShare = async () => {
    const url = `${window.location.origin}/posts/${postId}`;
    try {
      await navigator.clipboard.writeText(url);
      setShareState("copied");
    } catch {
      setShareState("failed");
    }
    setTimeout(() => setShareState("idle"), 2000);
  };

  const onFollowToggle = () => {
    if (isFollowing) unfollow.mutate(authorUserId);
    else follow.mutate(authorUserId);
  };

  const onConfirmBlock = () => {
    block.mutate(
      { target_id: authorUserId },
      { onSuccess: () => setBlockOpen(false) },
    );
  };

  return (
    <>
      <RxMenu.Root open={open} onOpenChange={setOpen}>
        <RxMenu.Trigger asChild>
          <button
            type="button"
            aria-label="More actions"
            className="inline-flex items-center justify-center size-8 rounded-full border hover:bg-white/5 outline-none focus:ring-2 focus:ring-[var(--season-accent)]/40"
            style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
          >
            <MoreHorizontal className="size-4" />
          </button>
        </RxMenu.Trigger>
        <RxMenu.Portal>
          <RxMenu.Content
            align="end"
            sideOffset={6}
            className="z-50 min-w-[200px] rounded-xl border shadow-lg p-1 outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
            style={{
              backgroundColor: "#121814",
              borderColor: "#1F2A24",
              color: "#ECEFEA",
            }}
          >
            {authorUsername && (
              <MenuItem
                onSelect={onFollowToggle}
                disabled={followBusy}
                icon={
                  followBusy ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : isFollowing ? (
                    <UserMinus className="size-4" />
                  ) : (
                    <UserPlus className="size-4" />
                  )
                }
              >
                {isFollowing ? "Unfollow" : "Follow"}
              </MenuItem>
            )}
            <MenuItem
              onSelect={onShare}
              icon={
                shareState === "copied" ? (
                  <Check className="size-4" />
                ) : (
                  <Share2 className="size-4" />
                )
              }
            >
              {shareState === "copied"
                ? "Link copied"
                : shareState === "failed"
                  ? "Copy failed"
                  : "Share"}
            </MenuItem>
            <RxMenu.Separator
              className="my-1 h-px"
              style={{ backgroundColor: "#1F2A24" }}
            />
            <MenuItem
              onSelect={() => setBlockOpen(true)}
              danger
              icon={<Ban className="size-4" />}
            >
              Block
            </MenuItem>
            <MenuItem
              onSelect={() => setReportOpen(true)}
              danger
              icon={
                reported ? (
                  <Check className="size-4" />
                ) : (
                  <Flag className="size-4" />
                )
              }
            >
              {reported ? "Reported" : "Report"}
            </MenuItem>
          </RxMenu.Content>
        </RxMenu.Portal>
      </RxMenu.Root>

      {blockOpen && (
        <Modal title={`Block ${authorName}`} onClose={() => setBlockOpen(false)}>
          <div className="flex flex-col gap-4">
            <p className="text-sm" style={{ color: "#B7C0B8" }}>
              You won&apos;t see their posts and they won&apos;t be able to
              message you. You can undo this later from settings.
            </p>
            {block.error && (
              <p className="text-sm" style={{ color: "#FCA5A5" }}>
                {errorMessage(block.error)}
              </p>
            )}
            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setBlockOpen(false)}
                disabled={block.isPending}
                className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
                style={{ color: "#8B9A8E" }}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={onConfirmBlock}
                disabled={block.isPending}
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
                style={{
                  backgroundColor: "rgba(220,38,38,0.15)",
                  color: "#FCA5A5",
                  border: "1px solid rgba(220,38,38,0.35)",
                }}
              >
                {block.isPending && (
                  <Loader2 className="size-3.5 animate-spin" />
                )}
                Block
              </button>
            </div>
          </div>
        </Modal>
      )}

      {reportOpen && (
        <ReportModal
          postId={postId}
          onClose={() => setReportOpen(false)}
          onSubmitted={() => setReported(true)}
        />
      )}
    </>
  );
}

function ReportModal({
  postId,
  onClose,
  onSubmitted,
}: {
  postId: string;
  onClose: () => void;
  onSubmitted: () => void;
}) {
  const report = useSubmitReport();
  const form = useForm<CreateReportInput>({
    resolver: zodResolver(createReportSchema),
    defaultValues: {
      subject_type: "post",
      subject_id: postId,
      reason: "",
      description: "",
    },
  });

  const onSubmit = (v: CreateReportInput) => {
    report.mutate(
      {
        subject_type: "post",
        subject_id: postId,
        reason: v.reason.trim(),
        description: v.description?.trim() || undefined,
      },
      {
        onSuccess: () => {
          onSubmitted();
          onClose();
        },
      },
    );
  };

  const submitError = errorMessage(report.error);

  return (
    <Modal title="Report post" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<CreateReportInput>
            name="reason"
            label="Reason"
            placeholder="e.g. spam, harassment"
            maxLength={32}
            hint="Short label, max 32 characters"
          />
          <ControlledTextarea<CreateReportInput>
            name="description"
            label="Details (optional)"
            placeholder="Add any context that helps moderators…"
            rows={4}
            maxLength={1000}
          />

          {submitError && (
            <p className="text-sm" style={{ color: "#FCA5A5" }}>
              {submitError}
            </p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={report.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#8B9A8E" }}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={report.isPending}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
              style={{
                backgroundColor: "var(--season-button)",
                color: "#0B100D",
              }}
            >
              {report.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              Submit report
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}

function MenuItem({
  onSelect,
  disabled,
  danger,
  icon,
  children,
}: {
  onSelect: () => void;
  disabled?: boolean;
  danger?: boolean;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <RxMenu.Item
      disabled={disabled}
      onSelect={() => onSelect()}
      className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm outline-none cursor-pointer data-[highlighted]:bg-white/5 data-[disabled]:opacity-60 data-[disabled]:cursor-not-allowed"
      style={{ color: danger ? "#FCA5A5" : "#ECEFEA" }}
    >
      {icon}
      <span>{children}</span>
    </RxMenu.Item>
  );
}
