import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { TripDetail } from "@/components/trips/trip-detail";

export const metadata: Metadata = {
  title: "Trip — TripCraft",
};

export default async function TripDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <PageShell back={{ href: "/trips", label: "All trips" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-5xl mx-auto">
        <TripDetail tripId={id} />
      </div>
    </PageShell>
  );
}
