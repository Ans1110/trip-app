"use client";

import { Modal } from "@/components/trips/modal";
import { ProfileSettingsForm } from "./profile-settings-form";

export function EditProfileDialog({ onClose }: { onClose: () => void }) {
  return (
    <Modal title="Edit profile" onClose={onClose} size="3xl">
      <ProfileSettingsForm onSaved={onClose} />
    </Modal>
  );
}
