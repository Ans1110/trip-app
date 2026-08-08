"use client";

import { Modal } from "@/components/trips/modal";
import { ProfileSettingsForm } from "./profile-settings-form";
import { MFASettings } from "./mfa-settings";

export function EditProfileDialog({ onClose }: { onClose: () => void }) {
  return (
    <Modal title="Edit profile" onClose={onClose} size="3xl">
      <div className="flex flex-col gap-8">
        <ProfileSettingsForm onSaved={onClose} />
        <div
          className="pt-6"
          style={{ borderTop: "1px solid #1F2A24" }}
        >
          <h2
            className="text-xs tracking-[0.2em] uppercase mb-3"
            style={{ color: "#8B9A8E" }}
          >
            Security
          </h2>
          <MFASettings />
        </div>
      </div>
    </Modal>
  );
}
