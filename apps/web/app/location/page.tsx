import type { Metadata } from "next";

import { PlacesView } from "@/components/location/places-view";
import { PageShell } from "@/components/trips/page-shell";

export const metadata: Metadata = {
  title: "Places — TripCraft",
};

export default function LocationPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="relative flex-1 min-h-[560px]">
        <PlacesView />
      </div>
    </PageShell>
  );
}
