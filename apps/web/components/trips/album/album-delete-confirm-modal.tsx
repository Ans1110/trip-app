"use client";

import { Loader2 } from "lucide-react";

import { Modal } from "@/components/trips/modal";
import { errorMessage, useDeletePhoto } from "@/hooks/album-hooks";
import type { AlbumPhoto } from "@/lib/apis/album-api";

export function AlbumDeleteConfirmModal({
  photo,
  onClose,
}: {
  photo: AlbumPhoto;
  onClose: () => void;
}) {
  const del = useDeletePhoto();

  const onConfirm = () => {
    del.mutate(
      { photoId: photo.id, tripId: photo.trip_id },
      { onSuccess: () => onClose() },
    );
  };

  const submitError = errorMessage(del.error);

  return (
    <Modal title="Delete photo?" onClose={onClose}>
      <div className="flex flex-col gap-4">
        <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
          This photo will be removed from the album for everyone on this trip.
          This action can&apos;t be undone.
        </p>

        {submitError && <p className="text-sm text-destructive">{submitError}</p>}

        <div className="flex items-center justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={del.isPending}
            className="px-4 py-2 text-sm rounded-full hover:bg-muted disabled:opacity-60 text-muted-foreground"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={del.isPending}
            className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{ backgroundColor: "var(--destructive)", color: "var(--destructive)" }}
          >
            {del.isPending && <Loader2 className="size-3.5 animate-spin" />}
            Delete
          </button>
        </div>
      </div>
    </Modal>
  );
}
