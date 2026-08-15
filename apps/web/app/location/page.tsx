import type { Metadata } from "next";

import { LocationView } from "@/components/location/location-view";
import { PageShell } from "@/components/trips/page-shell";

export const metadata: Metadata = {
  title: "Places — TripCraft",
};

export default function LocationPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-7xl mx-auto">
        <LocationView />
      </div>
    </PageShell>
  );
}
