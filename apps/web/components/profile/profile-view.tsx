"use client";

import { useEffect, useRef, useState } from "react";
import {
  Check,
  Loader2,
  Plus,
  Settings,
  Share2,
  UserMinus,
  UserPlus,
  X,
} from "lucide-react";

import {
  errorMessage,
  useFollow,
  useProfileByUsername,
  useUnfollow,
  useUpdateMyProfile,
} from "@/hooks/profile-hooks";
import { EditProfileDialog } from "./edit-profile-dialog";
import { FollowersModal, type FollowersTab } from "./followers-modal";
import { UserPosts } from "./user-posts";

const MAX_TAGS = 20;
const MAX_TAG_LEN = 32;

export function ProfileView({ username }: { username: string }) {
  const { data: profile, isLoading, error } = useProfileByUsername(username);
  const follow = useFollow();
  const unfollow = useUnfollow();
  const [editOpen, setEditOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [followersTab, setFollowersTab] = useState<FollowersTab | null>(null);

  const shareProfile = async () => {
    if (!profile) return;
    const url = `${window.location.origin}/profile/${profile.username}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading profile
      </div>
    );
  }
  if (error || !profile) {
    return (
      <p className="text-sm" style={{ color: "#FCA5A5" }}>
        {errorMessage(error) ?? "Profile not found"}
      </p>
    );
  }

  const initial = (profile.username || profile.name || "U")
    .trim()[0]!
    .toUpperCase();
  const isFollowing = profile.is_following;
  const pending = follow.isPending || unfollow.isPending;

  return (
    <div>
      <div
        className="h-40 md:h-56 rounded-2xl overflow-hidden mb-0.5"
        style={{
          background: profile.cover_image
            ? `url(${profile.cover_image}) center/cover no-repeat`
            : "linear-gradient(135deg, #1E2A2E 0%, #0A1014 100%)",
          border: "1px solid #1F2A24",
        }}
        aria-hidden={!profile.cover_image}
      />

      <div className="relative z-10 flex flex-col md:flex-row md:items-start gap-5 md:gap-6 px-2 md:px-4">
        <div className="flex flex-col items-center gap-2 -mt-12 md:-mt-14 shrink-0">
          <div
            className="size-24 md:size-28 rounded-full overflow-hidden inline-flex items-center justify-center text-2xl font-semibold ring-4"
            style={{
              backgroundColor: profile.avatar_url
                ? "transparent"
                : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
              color: "#ECEFEA",
              boxShadow: "0 8px 24px rgba(0,0,0,0.35)",
            }}
          >
            {profile.avatar_url ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={profile.avatar_url}
                alt={profile.name}
                className="size-full object-cover"
              />
            ) : (
              <span aria-hidden>{initial}</span>
            )}
          </div>
          <SocialLinks
            instagram={profile.social_instagram}
            x={profile.social_x}
            youtube={profile.social_youtube}
            tiktok={profile.social_tiktok}
          />
        </div>

        <div className="flex-1 min-w-0 flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h1
              className="text-2xl font-semibold truncate"
              style={{
                fontFamily: "var(--font-display, Georgia, serif)",
                color: "#ECEFEA",
              }}
            >
              {profile.name || profile.username}
            </h1>
            <p className="text-sm mt-0.5" style={{ color: "#8B9A8E" }}>
              @{profile.username}
            </p>
            <TravelTags tags={profile.travel_tags} editable={profile.is_self} />
          </div>
          <div className="flex flex-col items-end gap-2 shrink-0 mt-1">
            <div className="flex gap-4 text-sm">
              <button
                type="button"
                onClick={() => setFollowersTab("followers")}
                className="text-right rounded-lg px-2 py-1 hover:bg-white/5"
              >
                <span
                  style={{ color: "#8B9A8E" }}
                  className="block text-2xs tracking-wide"
                >
                  Followers
                </span>
                <span
                  className="block mt-1 text-lg font-semibold"
                  style={{ color: "#ECEFEA" }}
                >
                  {profile.followers_count}
                </span>
              </button>
              <button
                type="button"
                onClick={() => setFollowersTab("following")}
                className="text-right rounded-lg px-2 py-1 hover:bg-white/5"
              >
                <span
                  style={{ color: "#8B9A8E" }}
                  className="block text-2xs tracking-wide"
                >
                  Following
                </span>
                <span
                  className="block mt-1 text-lg font-semibold"
                  style={{ color: "#ECEFEA" }}
                >
                  {profile.following_count}
                </span>
              </button>
            </div>
            <div className="flex items-center gap-2 self-stretch">
              {profile.is_self ? (
                <button
                  type="button"
                  onClick={() => setEditOpen(true)}
                  className="flex-1 basis-0 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-full text-sm border hover:bg-white/5"
                  style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
                >
                  <Settings className="size-4" />
                  Edit
                </button>
              ) : (
                <button
                  type="button"
                  disabled={pending}
                  onClick={() =>
                    isFollowing
                      ? unfollow.mutate(profile.user_id)
                      : follow.mutate(profile.user_id)
                  }
                  className="season-transition flex-1 basis-0 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-full text-sm font-medium disabled:opacity-60"
                  style={{
                    backgroundColor: isFollowing
                      ? "transparent"
                      : "var(--season-button)",
                    color: isFollowing ? "#ECEFEA" : "#0B100D",
                    border: isFollowing ? "1px solid #1F2A24" : "none",
                  }}
                >
                  {pending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : isFollowing ? (
                    <>
                      <UserMinus className="size-4" />
                      Unfollow
                    </>
                  ) : (
                    <>
                      <UserPlus className="size-4" />
                      Follow
                    </>
                  )}
                </button>
              )}
              <button
                type="button"
                onClick={shareProfile}
                aria-label={copied ? "Profile link copied" : "Share profile"}
                className="flex-1 basis-0 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-full text-sm border hover:bg-white/5"
                style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
              >
                {copied ? (
                  <>
                    <Check className="size-4" />
                    Copied
                  </>
                ) : (
                  <>
                    <Share2 className="size-4" />
                    Share
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>

      {profile.bio && <Bio text={profile.bio} />}

      <UserPosts userId={profile.user_id} />

      {editOpen && <EditProfileDialog onClose={() => setEditOpen(false)} />}
      {followersTab && (
        <FollowersModal
          userId={profile.user_id}
          initialTab={followersTab}
          followersCount={profile.followers_count}
          followingCount={profile.following_count}
          onClose={() => setFollowersTab(null)}
        />
      )}
    </div>
  );
}

const BIO_DESKTOP_LIMIT = 60;
const BIO_MOBILE_LIMIT = 30;

function Bio({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const [isDesktop, setIsDesktop] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(min-width: 768px)");
    const update = () => setIsDesktop(mq.matches);
    update();
    mq.addEventListener("change", update);
    return () => mq.removeEventListener("change", update);
  }, []);
  const limit = isDesktop ? BIO_DESKTOP_LIMIT : BIO_MOBILE_LIMIT;
  const words = text.trim().split(/\s+/);
  const isLong = words.length > limit;
  const preview = isLong ? `${words.slice(0, limit).join(" ")}…` : text;
  return (
    <div className="mt-6 max-w-2xl">
      <p
        className="text-[15px] leading-relaxed whitespace-pre-wrap"
        style={{ color: "#ECEFEA" }}
      >
        {expanded || !isLong ? text : preview}
      </p>
      {isLong && (
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="mt-1 text-xs font-medium hover:underline"
          style={{ color: "var(--season-accent)" }}
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      )}
    </div>
  );
}

const SOCIAL_ICONS: Array<{
  key: "instagram" | "x" | "youtube" | "tiktok";
  src: string;
  label: string;
}> = [
  {
    key: "instagram",
    src: "/ins.svg",
    label: "Instagram",
  },
  { key: "x", src: "/x.svg", label: "X" },
  {
    key: "youtube",
    src: "/youtube.svg",
    label: "YouTube",
  },
  { key: "tiktok", src: "/tik-tok.svg", label: "TikTok" },
];

function SocialLinks({
  instagram,
  x,
  youtube,
  tiktok,
}: {
  instagram?: string;
  x?: string;
  youtube?: string;
  tiktok?: string;
}) {
  const links: Record<string, string | undefined> = {
    instagram,
    x,
    youtube,
    tiktok,
  };
  const items = SOCIAL_ICONS.filter((s) => links[s.key]?.trim());
  if (items.length === 0) return null;
  return (
    <ul className="flex items-center gap-2">
      {items.map((s) => (
        <li key={s.key}>
          <a
            href={links[s.key]}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={s.label}
            className="inline-flex items-center justify-center size-4 opacity-80 hover:opacity-100 transition-opacity"
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={s.src} alt="" className="size-full" />
          </a>
        </li>
      ))}
    </ul>
  );
}

function TravelTags({ tags, editable }: { tags: string[]; editable: boolean }) {
  const update = useUpdateMyProfile();
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (adding) inputRef.current?.focus();
  }, [adding]);

  const commit = () => {
    const value = draft.trim();
    setDraft("");
    setAdding(false);
    if (!value) return;
    if (tags.some((t) => t.toLowerCase() === value.toLowerCase())) return;
    if (tags.length >= MAX_TAGS) return;
    const next = [...tags, value.slice(0, MAX_TAG_LEN)];
    update.mutate({ travel_tags: next });
  };

  const remove = (tag: string) => {
    const next = tags.filter((t) => t !== tag);
    update.mutate({ travel_tags: next });
  };

  if (!editable && tags.length === 0) return null;

  return (
    <ul className="mt-3 flex flex-wrap gap-1.5 items-center">
      {tags.map((tag) => (
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
          {editable && (
            <button
              type="button"
              onClick={() => remove(tag)}
              disabled={update.isPending}
              aria-label={`Remove ${tag}`}
              className="inline-flex items-center justify-center size-4 rounded-full hover:bg-white/10 disabled:opacity-60"
            >
              <X className="size-3" />
            </button>
          )}
        </li>
      ))}
      {editable &&
        tags.length < MAX_TAGS &&
        (adding ? (
          <li>
            <input
              ref={inputRef}
              value={draft}
              onChange={(e) => setDraft(e.target.value.slice(0, MAX_TAG_LEN))}
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
              disabled={update.isPending}
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
              disabled={update.isPending}
              className="inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#8B9A8E", border: "1px dashed #1F2A24" }}
            >
              {update.isPending ? (
                <Loader2 className="size-3 animate-spin" />
              ) : (
                <Plus className="size-3" />
              )}
              Add tag
            </button>
          </li>
        ))}
    </ul>
  );
}
