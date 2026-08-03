"use client";

import { useEffect, useRef, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ImagePlus, Loader2, Save, Trash2, Upload } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import {
  errorMessage,
  useMyProfile,
  useUpdateMyProfile,
} from "@/hooks/profile-hooks";
import { uploadImage } from "@/lib/cloudinary-upload";
import {
  updateProfileSchema,
  type UpdateProfileFormInput,
} from "@/lib/schemas/profile-schemas";

const LABEL = "text-[11px] tracking-[0.2em] uppercase";
const INPUT = "px-4 py-3";
const MAX_BYTES = 8 * 1024 * 1024;
const ACCEPTED = ["image/jpeg", "image/png", "image/webp", "image/gif"];

export function ProfileSettingsForm({ onSaved }: { onSaved?: () => void } = {}) {
  const { data: profile, isLoading } = useMyProfile();
  const mutation = useUpdateMyProfile({
    onSuccess: () => onSaved?.(),
  });

  const form = useForm<UpdateProfileFormInput>({
    resolver: zodResolver(updateProfileSchema),
    defaultValues: {
      username: "",
      bio: "",
      avatar_url: "",
      cover_image: "",
      social_instagram: "",
      social_x: "",
      social_youtube: "",
      social_tiktok: "",
    },
  });

  useEffect(() => {
    if (!profile) return;
    form.reset({
      username: profile.username ?? "",
      bio: profile.bio ?? "",
      avatar_url: profile.avatar_url ?? "",
      cover_image: profile.cover_image ?? "",
      social_instagram: profile.social_instagram ?? "",
      social_x: profile.social_x ?? "",
      social_youtube: profile.social_youtube ?? "",
      social_tiktok: profile.social_tiktok ?? "",
    });
  }, [profile, form]);

  const submitError = errorMessage(mutation.error);
  const avatarUrl = form.watch("avatar_url") ?? "";
  const coverUrl = form.watch("cover_image") ?? "";

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm" style={{ color: "#8B9A8E" }}>
        <Loader2 className="size-4 animate-spin" />
        Loading profile
      </div>
    );
  }

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit((values) => {
          mutation.mutate(values);
        })}
        className="grid grid-cols-1 md:grid-cols-2 gap-5"
        noValidate
      >
        <CoverImageField
          value={coverUrl}
          onChange={(url) =>
            form.setValue("cover_image", url, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        />

        <AvatarField
          value={avatarUrl}
          fallbackLetter={
            (profile?.username || profile?.name || "U").trim()[0]!.toUpperCase()
          }
          onChange={(url) =>
            form.setValue("avatar_url", url, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        />

        <div className="md:col-span-2 grid grid-cols-1 md:grid-cols-2 gap-5">
          <ControlledInput<UpdateProfileFormInput>
            name="username"
            label="Username"
            placeholder="jane_doe"
            autoComplete="username"
            labelClassName={LABEL}
            className={INPUT}
            hint="3–32 chars, letters, numbers, underscore"
          />
          <ControlledTextarea<UpdateProfileFormInput>
            name="bio"
            label="Bio"
            placeholder="A little about you"
            labelClassName={LABEL}
            className="px-4 py-3 min-h-[96px] max-h-40 chat-scroll"
            containerClassName="md:col-span-2"
            hint="Up to 500 characters"
          />
          <ControlledInput<UpdateProfileFormInput>
            name="social_instagram"
            label="Instagram URL"
            placeholder="https://instagram.com/…"
            labelClassName={LABEL}
            className={INPUT}
          />
          <ControlledInput<UpdateProfileFormInput>
            name="social_x"
            label="X (Twitter) URL"
            placeholder="https://x.com/…"
            labelClassName={LABEL}
            className={INPUT}
          />
          <ControlledInput<UpdateProfileFormInput>
            name="social_youtube"
            label="YouTube URL"
            placeholder="https://youtube.com/…"
            labelClassName={LABEL}
            className={INPUT}
          />
          <ControlledInput<UpdateProfileFormInput>
            name="social_tiktok"
            label="TikTok URL"
            placeholder="https://tiktok.com/…"
            labelClassName={LABEL}
            className={INPUT}
          />
        </div>

        <div className="md:col-span-2 flex items-center gap-3">
          <button
            type="submit"
            disabled={mutation.isPending}
            className="season-transition inline-flex items-center gap-2 px-5 py-3 rounded-full text-sm font-medium hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
              boxShadow: "0 8px 24px var(--season-button-shadow)",
            }}
          >
            {mutation.isPending ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Saving
              </>
            ) : (
              <>
                <Save className="size-4" />
                Save changes
              </>
            )}
          </button>
          {mutation.isSuccess && !mutation.isPending && (
            <span className="text-xs" style={{ color: "#8B9A8E" }}>
              Saved
            </span>
          )}
        </div>

        {submitError && (
          <p
            className="md:col-span-2 text-sm rounded-lg px-3 py-2"
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
      </form>
    </FormProvider>
  );
}

function useImagePicker(onUploaded: (url: string) => void) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const openPicker = () => inputRef.current?.click();

  const handleFile = async (file: File) => {
    setError(null);
    if (!ACCEPTED.includes(file.type)) {
      setError("Use a JPEG, PNG, WebP, or GIF image.");
      return;
    }
    if (file.size > MAX_BYTES) {
      setError("Image must be 8 MB or smaller.");
      return;
    }
    setBusy(true);
    try {
      const uploaded = await uploadImage(file);
      onUploaded(uploaded.secure_url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed.");
    } finally {
      setBusy(false);
    }
  };

  return { inputRef, busy, error, openPicker, handleFile };
}

function CoverImageField({
  value,
  onChange,
}: {
  value: string;
  onChange: (url: string) => void;
}) {
  const { inputRef, busy, error, openPicker, handleFile } =
    useImagePicker(onChange);

  return (
    <div className="md:col-span-2 flex flex-col gap-2">
      <span className={LABEL} style={{ color: "#8B9A8E" }}>
        Cover image
      </span>
      <div
        className="relative rounded-2xl overflow-hidden"
        style={{
          border: "1px solid #1F2A24",
          aspectRatio: "16 / 6",
          background: value
            ? `url(${value}) center/cover no-repeat`
            : "linear-gradient(135deg, #1E2A2E 0%, #0A1014 100%)",
        }}
      >
        {!value && (
          <div
            className="absolute inset-0 flex flex-col items-center justify-center gap-2"
            style={{ color: "#6B7A6F" }}
          >
            <ImagePlus className="size-6" />
            <p className="text-xs">No cover yet</p>
          </div>
        )}
        <div className="absolute inset-x-0 bottom-0 p-3 flex items-center justify-end gap-2">
          {value && (
            <button
              type="button"
              onClick={() => onChange("")}
              disabled={busy}
              aria-label="Remove cover"
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full backdrop-blur-sm hover:bg-white/10 disabled:opacity-60"
              style={{
                color: "#FCA5A5",
                border: "1px solid rgba(255,255,255,0.15)",
                backgroundColor: "rgba(0,0,0,0.4)",
              }}
            >
              <Trash2 className="size-3.5" />
              Remove
            </button>
          )}
          <button
            type="button"
            onClick={openPicker}
            disabled={busy}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full backdrop-blur-sm hover:bg-white/10 disabled:opacity-60"
            style={{
              color: "#ECEFEA",
              border: "1px solid rgba(255,255,255,0.15)",
              backgroundColor: "rgba(0,0,0,0.4)",
            }}
          >
            {busy ? (
              <>
                <Loader2 className="size-3.5 animate-spin" />
                Uploading
              </>
            ) : (
              <>
                <Upload className="size-3.5" />
                {value ? "Replace" : "Upload"}
              </>
            )}
          </button>
        </div>
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED.join(",")}
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void handleFile(file);
          e.target.value = "";
        }}
      />
      {error ? (
        <p className="text-[11px]" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      ) : (
        <p className="text-[11px]" style={{ color: "#6B7A6F" }}>
          JPEG, PNG, WebP, or GIF. Max 8 MB.
        </p>
      )}
    </div>
  );
}

function AvatarField({
  value,
  fallbackLetter,
  onChange,
}: {
  value: string;
  fallbackLetter: string;
  onChange: (url: string) => void;
}) {
  const { inputRef, busy, error, openPicker, handleFile } =
    useImagePicker(onChange);

  return (
    <div className="md:col-span-2 flex items-center gap-5">
      <div
        className="size-20 rounded-full overflow-hidden inline-flex items-center justify-center text-2xl font-semibold shrink-0"
        style={{
          backgroundColor: value
            ? "transparent"
            : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
          color: "#ECEFEA",
          border: "1px solid #1F2A24",
        }}
      >
        {value ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={value}
            alt="Avatar preview"
            className="size-full object-cover"
          />
        ) : (
          <span aria-hidden>{fallbackLetter}</span>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <span className={LABEL} style={{ color: "#8B9A8E" }}>
          Avatar
        </span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={openPicker}
            disabled={busy}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
          >
            {busy ? (
              <>
                <Loader2 className="size-3.5 animate-spin" />
                Uploading
              </>
            ) : (
              <>
                <Upload className="size-3.5" />
                {value ? "Replace" : "Upload"}
              </>
            )}
          </button>
          {value && (
            <button
              type="button"
              onClick={() => onChange("")}
              disabled={busy}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#FCA5A5", border: "1px solid #1F2A24" }}
            >
              <Trash2 className="size-3.5" />
              Remove
            </button>
          )}
        </div>
        {error ? (
          <p className="text-[11px]" style={{ color: "#FCA5A5" }}>
            {error}
          </p>
        ) : (
          <p className="text-[11px]" style={{ color: "#6B7A6F" }}>
            Square image works best. Max 8 MB.
          </p>
        )}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED.join(",")}
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void handleFile(file);
          e.target.value = "";
        }}
      />
    </div>
  );
}
