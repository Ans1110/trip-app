import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { RoomPreview } from "@/components/trips/room-preview";

export const metadata: Metadata = {
  title: "Join trip — TripCraft",
};

export default async function RoomPreviewPage({
  params,
}: {
  params: Promise<{ code: string }>;
}) {
  const { code } = await params;
  return (
    <PageShell back={{ href: "/trips", label: "Your trips" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-2xl mx-auto">
        <RoomPreview code={code} />
      </div>
    </PageShell>
  );
}
