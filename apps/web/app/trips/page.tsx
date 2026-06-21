import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { TripsList } from "@/components/trips/trips-list";

export const metadata: Metadata = {
  title: "Trips — TripCraft",
};

export default function TripsPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-6xl mx-auto">
        <TripsList />
      </div>
    </PageShell>
  );
}
