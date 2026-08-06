"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ImagePlus, Loader2, Plus, Trash2, Upload, X } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledTextarea } from "@/components/ui/controlled-textarea";
import { errorMessage, useCreatePost, useUpdatePost } from "@/hooks/post-hooks";
import type { Post, PostStatus } from "@/lib/apis/post-api";
import { uploadImage } from "@/lib/cloudinary-upload";
import {
  createPostSchema,
  POST_TAGS_MAX,
  POST_TAG_MAX,
  type CreatePostFormInput,
} from "@/lib/schemas/post-schemas";

const COVER_MAX_BYTES = 8 * 1024 * 1024;
const COVER_ACCEPTED = ["image/jpeg", "image/png", "image/webp", "image/gif"];

type Props =
  | { mode: "create"; initial?: undefined }
  | { mode: "edit"; initial: Post };

export function PostForm(props: Props) {
  const router = useRouter();
  const isEdit = props.mode === "edit";
  const initial = isEdit ? props.initial : undefined;

  const create = useCreatePost();
  const update = useUpdatePost(initial?.id ?? "");

  const form = useForm<CreatePostFormInput>({
    resolver: zodResolver(createPostSchema),
    defaultValues: {
      title: initial?.title ?? "",
      content: initial?.content ?? "",
      cover_image: initial?.cover_image ?? "",
      tags: initial?.tags ?? [],
      status: initial?.status ?? "published",
    },
  });

  const tags = form.watch("tags") ?? [];
  const coverUrl = form.watch("cover_image") ?? "";
  const status = (form.watch("status") ?? "published") as PostStatus;
  const pending = create.isPending || update.isPending;
  const submitError = errorMessage(create.error) ?? errorMessage(update.error);

  const onSubmit = (v: CreatePostFormInput) => {
    const payload = {
      title: v.title.trim(),
      content: v.content,
      cover_image: v.cover_image?.trim() || undefined,
      tags: v.tags && v.tags.length > 0 ? v.tags : undefined,
      status: (v.status ?? "published") as PostStatus,
    };
    if (isEdit && initial) {
      update.mutate(payload, {
        onSuccess: (post) => router.push(`/posts/${post.id}`),
      });
    } else {
      create.mutate(payload, {
        onSuccess: (post) => router.push(`/posts/${post.id}`),
      });
    }
  };

  const submitLabel =
    status === "draft"
      ? isEdit
        ? "Save draft"
        : "Save as draft"
      : status === "archived"
        ? "Archive"
        : isEdit
          ? "Save changes"
          : "Publish";

  return (
    <FormProvider {...form}>
      <form
        className="flex flex-col gap-5"
        onSubmit={form.handleSubmit(onSubmit)}
        noValidate
      >
        <ControlledInput<CreatePostFormInput>
          name="title"
          label="Title"
          placeholder="A memorable title…"
          maxLength={200}
        />

        <CoverImageField
          value={coverUrl}
          onChange={(url) =>
            form.setValue("cover_image", url, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        />

        <ControlledTextarea<CreatePostFormInput>
          name="content"
          label="Content"
          placeholder="Tell your story…"
          rows={12}
        />

        <TagsEditor
          value={tags}
          onChange={(next) =>
            form.setValue("tags", next, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        />

        <StatusPicker
          value={status}
          onChange={(next) =>
            form.setValue("status", next, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
          allowArchived={isEdit}
        />

        {submitError && (
          <p className="text-sm" style={{ color: "#FCA5A5" }}>
            {submitError}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={() => router.back()}
            disabled={pending}
            className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#8B9A8E" }}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={pending}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {pending && <Loader2 className="size-3.5 animate-spin" />}
            {submitLabel}
          </button>
        </div>
      </form>
    </FormProvider>
  );
}

function StatusPicker({
  value,
  onChange,
  allowArchived,
}: {
  value: PostStatus;
  onChange: (next: PostStatus) => void;
  allowArchived: boolean;
}) {
  const options: { key: PostStatus; label: string; hint: string }[] = [
    { key: "draft", label: "Draft", hint: "Only you can see it" },
    { key: "published", label: "Published", hint: "Visible to everyone" },
  ];
  if (allowArchived) {
    options.push({
      key: "archived",
      label: "Archived",
      hint: "Hidden from feeds",
    });
  }
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs font-medium" style={{ color: "#ECEFEA" }}>
        Visibility
      </label>
      <div className="flex flex-wrap gap-1.5">
        {options.map((opt) => {
          const active = opt.key === value;
          return (
            <button
              key={opt.key}
              type="button"
              onClick={() => onChange(opt.key)}
              aria-pressed={active}
              className="px-3 py-1.5 text-xs rounded-full season-transition"
              style={{
                backgroundColor: active
                  ? "color-mix(in srgb, var(--season-accent) 18%, transparent)"
                  : "transparent",
                color: active ? "var(--season-accent)" : "#8B9A8E",
                border: `1px solid ${active ? "var(--season-accent)" : "#1F2A24"}`,
              }}
            >
              {opt.label}
            </button>
          );
        })}
      </div>
      <p className="text-[11px]" style={{ color: "#6B7A6F" }}>
        {options.find((o) => o.key === value)?.hint ?? ""}
      </p>
    </div>
  );
}

function TagsEditor({
  value,
  onChange,
}: {
  value: string[];
  onChange: (next: string[]) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (adding) inputRef.current?.focus();
  }, [adding]);

  const commit = () => {
    const v = draft.trim();
    setDraft("");
    setAdding(false);
    if (!v) return;
    if (value.some((t) => t.toLowerCase() === v.toLowerCase())) return;
    if (value.length >= POST_TAGS_MAX) return;
    onChange([...value, v.slice(0, POST_TAG_MAX)]);
  };

  const remove = (tag: string) => onChange(value.filter((t) => t !== tag));

  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs font-medium" style={{ color: "#ECEFEA" }}>
        Tags
      </label>
      <ul className="flex flex-wrap gap-1.5 items-center">
        {value.map((tag) => (
          <li
            key={tag}
            className="inline-flex items-center gap-1 text-xs pl-2.5 pr-1 py-1 rounded-full"
            style={{
              backgroundColor:
                "color-mix(in srgb, var(--season-accent) 14%, transparent)",
              color: "var(--season-accent)",
            }}
          >
            <span>{tag}</span>
            <button
              type="button"
              onClick={() => remove(tag)}
              aria-label={`Remove ${tag}`}
              className="inline-flex items-center justify-center size-4 rounded-full hover:bg-white/10"
            >
              <X className="size-3" />
            </button>
          </li>
        ))}
        {value.length < POST_TAGS_MAX &&
          (adding ? (
            <li>
              <input
                ref={inputRef}
                value={draft}
                onChange={(e) =>
                  setDraft(e.target.value.slice(0, POST_TAG_MAX))
                }
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    commit();
                  } else if (e.key === "Escape") {
                    e.preventDefault();
                    setDraft("");
                    setAdding(false);
                  }
                }}
                onBlur={commit}
                placeholder="new tag"
                className="text-xs px-2.5 py-1 rounded-full outline-none"
                style={{
                  backgroundColor: "#121814",
                  color: "#ECEFEA",
                  border: "1px dashed #1F2A24",
                  minWidth: "6rem",
                }}
              />
            </li>
          ) : (
            <li>
              <button
                type="button"
                onClick={() => setAdding(true)}
                className="inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full hover:bg-white/5"
                style={{ color: "#8B9A8E", border: "1px dashed #1F2A24" }}
              >
                <Plus className="size-3" />
                Add tag
              </button>
            </li>
          ))}
      </ul>
    </div>
  );
}

function CoverImageField({
  value,
  onChange,
}: {
  value: string;
  onChange: (url: string) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const openPicker = () => inputRef.current?.click();

  const handleFile = async (file: File) => {
    setError(null);
    if (!COVER_ACCEPTED.includes(file.type)) {
      setError("Use a JPEG, PNG, WebP, or GIF image.");
      return;
    }
    if (file.size > COVER_MAX_BYTES) {
      setError("Image must be 8 MB or smaller.");
      return;
    }
    setBusy(true);
    try {
      const uploaded = await uploadImage(file);
      onChange(uploaded.secure_url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs font-medium" style={{ color: "#ECEFEA" }}>
        Cover image (optional)
      </label>
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
        accept={COVER_ACCEPTED.join(",")}
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
