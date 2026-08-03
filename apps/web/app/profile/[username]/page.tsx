import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { ProfileView } from "@/components/profile/profile-view";

export const metadata: Metadata = {
  title: "Profile — TripCraft",
};

type PageProps = { params: Promise<{ username: string }> };

export default async function ProfilePage({ params }: PageProps) {
  const { username } = await params;
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-4xl mx-auto">
        <ProfileView username={username} />
      </div>
    </PageShell>
  );
}
