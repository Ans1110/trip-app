"use client";

import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { Modal } from "@/components/trips/modal";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { errorMessage, useUpdatePhotoCaption } from "@/hooks/album-hooks";
import type { AlbumPhoto } from "@/lib/apis/album-api";
import {
  ALBUM_CAPTION_MAX,
  albumCaptionSchema,
  type AlbumCaptionFormInput,
} from "@/lib/schemas/album-schemas";

export function AlbumCaptionModal({
  photo,
  onClose,
}: {
  photo: AlbumPhoto;
  onClose: () => void;
}) {
  const update = useUpdatePhotoCaption();
  const form = useForm<AlbumCaptionFormInput>({
    resolver: zodResolver(albumCaptionSchema),
    defaultValues: { caption: photo.caption ?? "" },
  });

  const onSubmit = (v: AlbumCaptionFormInput) => {
    update.mutate(
      {
        photoId: photo.id,
        tripId: photo.trip_id,
        input: { caption: v.caption.trim() },
      },
      { onSuccess: () => onClose() },
    );
  };

  const submitError = errorMessage(update.error);

  return (
    <Modal title="Edit caption" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={form.handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledTextarea<AlbumCaptionFormInput>
            name="caption"
            label="Caption"
            placeholder="A few words about this photo…"
            rows={4}
            maxLength={ALBUM_CAPTION_MAX}
          />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={update.isPending}
              className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={update.isPending}
              className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {update.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              Save
            </button>
          </div>
        </form>
      </FormProvider>
    </Modal>
  );
}
