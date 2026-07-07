import type { Metadata } from "next";

import { FriendCalendarView } from "@/components/friends/friend-calendar-view";
import { PageShell } from "@/components/trips/page-shell";

export const metadata: Metadata = {
  title: "Friend calendar — TripCraft",
};

export default async function FriendCalendarPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-6xl mx-auto">
        <FriendCalendarView friendId={id} />
      </div>
    </PageShell>
  );
}
