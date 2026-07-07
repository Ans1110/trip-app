import type { Metadata } from "next";

import { CalendarView } from "@/components/calendar/calendar-view";
import { PageShell } from "@/components/trips/page-shell";

export const metadata: Metadata = {
  title: "Calendar — TripCraft",
};

export default function CalendarPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-6xl mx-auto">
        <CalendarView />
      </div>
    </PageShell>
  );
}
