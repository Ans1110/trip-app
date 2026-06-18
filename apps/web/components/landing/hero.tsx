"use client";

import Link from "next/link";
import Image from "next/image";
import { ArrowRight } from "lucide-react";

import { useSeason } from "@/components/season-provider";

export function Hero() {
  const { active, setActive, seasons, season } = useSeason();

  return (
    <>
      <div
        className="fixed inset-0 pointer-events-none"
        style={{ zIndex: 0 }}
        aria-hidden
      >
        {seasons.map((s, i) => (
          <div
            key={s.name}
            className="absolute inset-0 transition-opacity duration-[900ms] ease-out"
            style={{ opacity: i === active ? 1 : 0 }}
          >
            <Image
              src={s.image}
              alt=""
              fill
              sizes="100vw"
              quality={95}
              priority={i === 0}
              fetchPriority={i === 0 ? "high" : "auto"}
              className="object-cover"
            />
          </div>
        ))}
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(180deg, rgba(11,16,13,0.08) 0%, rgba(11,16,13,0.22) 60%, rgba(11,16,13,0.40) 100%)",
          }}
        />
      </div>

      <section
        className="relative min-h-[88vh] overflow-hidden"
        style={{ zIndex: 1 }}
      >
        <div
          className="season-transition absolute top-24 right-6 md:right-12 text-[11px] tracking-[0.25em] font-mono uppercase"
          style={{ color: "var(--season-accent)" }}
        >
          {season.label}
        </div>

        <div
          className="absolute inset-x-0 bottom-0 h-1/2 pointer-events-none"
          style={{
            background:
              "linear-gradient(to top, rgba(11,16,13,0.78) 0%, rgba(11,16,13,0.35) 50%, rgba(11,16,13,0) 100%)",
          }}
          aria-hidden
        />

        <div className="relative z-10 max-w-7xl mx-auto px-6 md:px-12 pt-40 md:pt-48 pb-20 md:pb-24 flex flex-col min-h-[88vh] justify-end">
          <h1
            className="season-transition leading-[0.95] tracking-tight mb-6"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(3.75rem, 9vw, 7.5rem)",
              color: "var(--season-heading)",
              textShadow: "0 2px 24px rgba(0,0,0,0.45)",
            }}
          >
            Wander
            <br />
            <em
              className="season-transition"
              style={{ color: "var(--season-accent)" }}
            >
              deliberately.
            </em>
          </h1>

          <p
            className="season-transition max-w-xl text-base md:text-lg leading-relaxed mb-12"
            style={{
              color: "var(--season-body)",
              textShadow: "0 1px 12px rgba(0,0,0,0.5)",
            }}
          >
            Plan trips the way you actually travel — flexibly, together,
            anywhere.
          </p>

          <div
            className="flex flex-wrap gap-x-7 gap-y-3 mb-6 text-xs md:text-sm tracking-[0.2em] uppercase font-medium"
            role="tablist"
            aria-label="Travel season"
          >
            {seasons.map((s, i) => {
              const isActive = i === active;
              return (
                <button
                  key={s.name}
                  type="button"
                  role="tab"
                  aria-selected={isActive}
                  onClick={() => setActive(i)}
                  className="season-transition relative pb-1.5 focus:outline-none focus-visible:ring-2 rounded-sm"
                  style={{
                    color: isActive ? "var(--season-accent)" : "#B5C2B8",
                  }}
                >
                  {s.name}
                  <span
                    className="season-transition absolute left-0 right-0 -bottom-0.5 h-px"
                    style={{
                      backgroundColor: "var(--season-accent)",
                      opacity: isActive ? 1 : 0,
                      transitionProperty:
                        "color, background-color, border-color, box-shadow, opacity",
                    }}
                    aria-hidden
                  />
                </button>
              );
            })}
          </div>

          <div className="flex items-center gap-6 flex-wrap mb-6">
            <p
              className="season-transition text-sm"
              style={{ color: "var(--season-body)" }}
            >
              <span
                className="season-transition"
                style={{ color: "var(--season-accent)" }}
              ></span>{" "}
              Traveling through the four seasons
            </p>
          </div>

          <div className="flex items-center gap-6 flex-wrap">
            <Link
              href="/sign-up"
              className="season-transition inline-flex items-center gap-2 px-6 py-3.5 rounded-full text-sm font-medium tracking-wide hover:-translate-y-0.5 active:translate-y-0"
              style={{
                backgroundColor: "var(--season-button)",
                color: "#0B100D",
                boxShadow: "0 8px 24px var(--season-button-shadow)",
              }}
            >
              Start planning <ArrowRight className="size-4" />
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
