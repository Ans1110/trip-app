import type { Metadata } from "next";

import { PageShell } from "@/components/trips/page-shell";
import { WeatherView } from "@/components/weather/weather-view";

export const metadata: Metadata = {
  title: "Weather — TripCraft",
};

export default function WeatherPage() {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <div className="px-6 md:px-12 py-10 w-full max-w-5xl mx-auto">
        <WeatherView />
      </div>
    </PageShell>
  );
}
