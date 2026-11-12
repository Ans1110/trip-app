import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, Check, Compass } from "lucide-react";

import { Hero } from "@/components/landing/hero";
import { Nav } from "@/components/landing/nav";
import { VerifyEmailBanner } from "@/components/auth/verify-email-banner";

export const metadata: Metadata = {
  title: "TripCraft — Plan Your Perfect Journey",
  description:
    "Plan itineraries, collaborate with friends, and discover hidden gems — all in one quiet workspace built for curious travelers.",
};

const cards = [
  {
    eyebrow: "Routes",
    title: "Drag, drop, done.",
    body: "Reshape a day in seconds. The plan keeps up while you change your mind.",
    from: "var(--mint)",
    to: "var(--cyan-soft)",
  },
  {
    eyebrow: "Companions",
    title: "Plan together, in real time.",
    body: "No more shared spreadsheets. Everyone sees the same itinerary, instantly.",
    from: "var(--pink)",
    to: "var(--sky-soft)",
  },
  {
    eyebrow: "Offline",
    title: "Works where signal doesn't.",
    body: "Train through Hokkaido, hike in Patagonia. Your plan travels with you.",
    from: "var(--sky)",
    to: "var(--mint-soft)",
  },
];

const itinerary = [
  { days: "Day 1–3",   city: "Tokyo",     spots: "Senso-ji, Shibuya Crossing", done: true },
  { days: "Day 4–7",   city: "Kyoto",     spots: "Fushimi Inari, Arashiyama",  done: true },
  { days: "Day 8–10",  city: "Osaka",     spots: "Dotonbori, Namba",           done: false },
  { days: "Day 11–14", city: "Hiroshima", spots: "Peace Memorial, Miyajima",   done: false },
];

const avatarColors = ["var(--mint)", "var(--sky)", "var(--pink)", "var(--cyan)"];

export default function Home() {
  return (
    <div className="min-h-screen relative text-foreground">
      <Nav />
      <VerifyEmailBanner />

      <Hero />

      <div className="bg-wash relative z-[1]">
      {/* ── 3 IMAGE-LED CARDS ── */}
      <section
        id="features"
        className="relative px-6 md:px-12 py-24 md:py-32"
      >
        <div className="max-w-7xl mx-auto">
          <div className="mb-16 max-w-2xl">
            <p
              className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-4"
              style={{ color: "var(--mint-ink)" }}
            >
              How it feels
            </p>
            <h2
              className="season-transition leading-[1.05] tracking-tight"
              style={{
                fontFamily: "var(--font-display), system-ui, sans-serif",
                fontSize: "clamp(2rem, 4vw, 3.5rem)",
                color: "var(--foreground)",
              }}
            >
              Built for the part of travel{" "}
              <em
                className="season-transition"
                style={{ color: "var(--pink-ink)" }}
              >
                before
              </em>{" "}
              the travel.
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-5 md:gap-6">
            {cards.map((c, i) => (
              <article
                key={i}
                className="group relative rounded-2xl overflow-hidden transition-[translate,box-shadow] duration-500 ease-out hover:-translate-y-1 hover:shadow-soft-lg"
                style={{ backgroundColor: "var(--card)" }}
              >
                <div
                  className="relative h-64 md:h-72 w-full overflow-hidden"
                >
                  <div
                    className="absolute inset-0 transition-transform duration-700 group-hover:scale-[1.04]"
                    style={{
                      background: `linear-gradient(135deg, ${c.from} 0%, ${c.to} 100%)`,
                    }}
                    aria-hidden
                  />
                  <div
                    className="absolute inset-0 pointer-events-none"
                    style={{
                      background:
                        "linear-gradient(to bottom, transparent 55%, color-mix(in oklch, var(--card) 70%, transparent) 100%)",
                    }}
                    aria-hidden
                  />
                </div>

                <div className="p-6 md:p-7">
                  <p
                    className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-3"
                    style={{ color: "var(--mint-ink)" }}
                  >
                    {c.eyebrow}
                  </p>
                  <p
                    className="leading-snug mb-3"
                    style={{
                      fontFamily: "var(--font-display), system-ui, sans-serif",
                      fontSize: "1.6rem",
                      color: "var(--foreground)",
                    }}
                  >
                    {c.title}
                  </p>
                  <p
                    className="text-sm leading-relaxed mb-6"
                    style={{ color: "var(--muted-foreground)" }}
                  >
                    {c.body}
                  </p>
                  <span
                    className="season-transition inline-flex items-center gap-1.5 text-xs tracking-[0.15em] uppercase font-medium"
                    style={{ color: "var(--pink-ink)" }}
                  >
                    Learn more <ArrowRight className="size-3.5" />
                  </span>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* ── ITINERARY SHOWCASE ── */}
      <section
        id="preview"
        className="relative px-6 md:px-12 py-24 md:py-32"
      >
        <div className="max-w-7xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-16 lg:gap-20 items-center">
          <div>
            <p
              className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-4"
              style={{ color: "var(--mint-ink)" }}
            >
              The workspace
            </p>
            <h2
              className="season-transition leading-[1.05] tracking-tight mb-6"
              style={{
                fontFamily: "var(--font-display), system-ui, sans-serif",
                fontSize: "clamp(2rem, 3.5vw, 3rem)",
                color: "var(--foreground)",
              }}
            >
              A workspace that thinks in{" "}
              <em
                className="season-transition"
                style={{ color: "var(--pink-ink)" }}
              >
                days
              </em>
              , not tabs.
            </h2>
            <p
              className="text-base leading-relaxed mb-8 max-w-md"
              style={{ color: "var(--muted-foreground)" }}
            >
              Drag a day to swap it. Drop a city on the map. TripCraft holds the
              itinerary while you keep changing your mind — which, if you&rsquo;re
              honest, is most of the planning.
            </p>
            <Link
              href="#"
              className="season-transition inline-flex items-center gap-2 text-sm tracking-[0.15em] uppercase font-medium border-b pb-0.5"
              style={{
                color: "var(--pink-ink)",
                borderColor: "var(--season-accent)",
              }}
            >
              Try the demo trip <ArrowRight className="size-3.5" />
            </Link>
          </div>

          <div className="flex justify-center lg:justify-end">
            <div
              className="w-full max-w-md rounded-2xl overflow-hidden"
              style={{
                backgroundColor: "var(--card)",
                boxShadow: "var(--shadow-soft-lg)",
              }}
            >
              <div
                className="px-6 pt-6 pb-4"
                style={{ borderBottom: "1px solid var(--border)" }}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xl" aria-hidden>🗾</span>
                    <span
                      className="font-semibold text-[15px]"
                      style={{ color: "var(--foreground)" }}
                    >
                      Japan · Spring 2026
                    </span>
                  </div>
                  <span
                    className="season-transition text-[11px] tracking-wide px-2.5 py-1 rounded-full font-semibold"
                    style={{
                      backgroundColor:
                        "color-mix(in srgb, var(--season-accent) 14%, transparent)",
                      color: "var(--pink-ink)",
                    }}
                  >
                    14 days
                  </span>
                </div>
                <p className="text-xs" style={{ color: "var(--subtle-foreground)" }}>
                  4 travelers · Tokyo · Kyoto · Osaka
                </p>
              </div>

              <div className="px-6 py-5 space-y-4">
                {itinerary.map((stop, i) => (
                  <div key={i} className="flex gap-3">
                    <div className="flex flex-col items-center pt-1 flex-shrink-0">
                      <div
                        className="season-transition size-2.5 rounded-full"
                        style={{
                          backgroundColor: stop.done
                            ? "var(--season-button)"
                            : "var(--input)",
                        }}
                      />
                      {i < itinerary.length - 1 && (
                        <div
                          className="season-transition w-px mt-1"
                          style={{
                            height: 24,
                            backgroundColor: stop.done
                              ? "var(--mint)"
                              : "var(--border)",
                          }}
                        />
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span
                          className="text-[11px] font-medium tabular-nums"
                          style={{ color: "var(--subtle-foreground)" }}
                        >
                          {stop.days}
                        </span>
                        <span
                          className="text-sm font-semibold"
                          style={{ color: "var(--foreground)" }}
                        >
                          {stop.city}
                        </span>
                        {stop.done && (
                          <Check
                            className="season-transition size-3.5"
                            style={{ color: "var(--mint-ink)" }}
                          />
                        )}
                      </div>
                      <p className="text-xs mt-0.5" style={{ color: "var(--subtle-foreground)" }}>
                        {stop.spots}
                      </p>
                    </div>
                  </div>
                ))}
              </div>

              <div
                className="px-6 py-4 flex items-center justify-between"
                style={{
                  backgroundColor: "var(--surface-sunken)",
                  borderTop: "1px solid var(--border)",
                }}
              >
                <div className="flex -space-x-1.5">
                  {avatarColors.map((color, i) => (
                    <div
                      key={i}
                      className="size-6 rounded-full"
                      style={{
                        backgroundColor: color,
                        boxShadow: "0 0 0 2px var(--card)",
                      }}
                    />
                  ))}
                </div>
                <div className="flex items-center gap-2">
                  <div
                    className="h-1.5 w-20 rounded-full overflow-hidden"
                    style={{ backgroundColor: "var(--muted)" }}
                  >
                    <div
                      className="season-transition h-full rounded-full"
                      style={{
                        width: "52%",
                        backgroundColor: "var(--season-accent)",
                      }}
                    />
                  </div>
                  <span className="text-xs" style={{ color: "var(--subtle-foreground)" }}>
                    52%
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ── CTA ── */}
      <section className="relative px-6 md:px-12 py-28">
        <div className="max-w-2xl mx-auto text-center">
          <div
            className="season-transition inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-[11px] font-medium tracking-[0.2em] uppercase mb-8"
            style={{
              backgroundColor:
                "color-mix(in srgb, var(--season-accent) 14%, transparent)",
              color: "var(--pink-ink)",
            }}
          >
            Free to start
          </div>

          <h2
            className="season-transition leading-[1.05] tracking-tight mb-6"
            style={{
              fontFamily: "var(--font-display), system-ui, sans-serif",
              fontSize: "clamp(2.5rem, 5vw, 4rem)",
              color: "var(--foreground)",
            }}
          >
            Your next adventure
            <br />
            <em
              className="season-transition"
              style={{ color: "var(--pink-ink)" }}
            >
              starts here.
            </em>
          </h2>

          <p className="text-lg mb-10" style={{ color: "var(--muted-foreground)" }}>
            Join 50,000+ travelers who use TripCraft to plan unforgettable
            journeys.
          </p>

          <Link
            href="/sign-up"
            className="season-transition inline-flex items-center justify-center gap-2 px-8 py-4 rounded-full text-base font-medium hover:-translate-y-0.5 active:translate-y-0"
            style={{
              backgroundColor: "var(--season-button)",
              color: "var(--primary-foreground)",
              boxShadow: "0 12px 32px var(--season-button-shadow)",
            }}
          >
            Create your first trip <ArrowRight className="size-4" />
          </Link>
        </div>
      </section>

      {/* ── FOOTER ── */}
      <footer
        className="relative px-6 md:px-12 py-10 border-t"
        style={{ borderColor: "var(--border)" }}
      >
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="flex items-center gap-2">
            <Compass
              className="season-transition size-4"
              style={{ color: "var(--mint-ink)" }}
            />
            <span
              className="text-lg"
              style={{
                fontFamily: "var(--font-display), system-ui, sans-serif",
                color: "var(--foreground)",
              }}
            >
              TripCraft
            </span>
          </div>

          <nav
            className="flex flex-wrap items-center justify-center gap-6 text-sm"
            style={{ color: "var(--muted-foreground)" }}
          >
            {["Routes", "Itinerary", "Blog", "Privacy", "Terms", "Contact"].map(
              (item) => (
                <Link
                  key={item}
                  href="#"
                  className="hover:text-foreground transition-colors"
                >
                  {item}
                </Link>
              ),
            )}
          </nav>

          <p className="text-sm" style={{ color: "var(--subtle-foreground)" }}>
            © 2026 TripCraft
          </p>
        </div>
      </footer>
      </div>
    </div>
  );
}
