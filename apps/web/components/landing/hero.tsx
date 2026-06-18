"use client";

import Link from "next/link";
import Image from "next/image";
import { useState } from "react";
import { ArrowRight } from "lucide-react";

type Season = {
  name: string;
  label: string;
  image: string;
};

const seasons: Season[] = [
  { name: "Spring", label: "MAR · APR · MAY", image: "/spring.jpeg" },
  { name: "Summer", label: "JUN · JUL · AUG", image: "/summer.jpeg" },
  { name: "Autumn", label: "SEP · OCT · NOV", image: "/autumn.jpeg" },
  { name: "Winter", label: "DEC · JAN · FEB", image: "/winter.jpeg" },
];

export function Hero() {
  const [active, setActive] = useState(0);
  const current = seasons[active];

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
              priority={i === 0}
              className="object-cover"
            />
          </div>
        ))}
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(180deg, rgba(11,16,13,0.15) 0%, rgba(11,16,13,0.35) 60%, rgba(11,16,13,0.55) 100%)",
          }}
        />
      </div>

      <section
        className="relative min-h-[88vh] overflow-hidden"
        style={{ zIndex: 1 }}
      >
        <div
          className="absolute top-24 right-6 md:right-12 text-[11px] tracking-[0.25em] font-mono uppercase transition-opacity duration-500"
          style={{ color: "#A8E0B4" }}
        >
          {current.label}
        </div>

        <div
          className="absolute inset-x-0 bottom-0 h-2/3 pointer-events-none"
          style={{
            background:
              "linear-gradient(to top, rgba(11,16,13,0.85) 0%, rgba(11,16,13,0.5) 40%, rgba(11,16,13,0) 100%)",
          }}
          aria-hidden
        />

        <div className="relative z-10 max-w-7xl mx-auto px-6 md:px-12 pt-40 md:pt-48 pb-20 md:pb-24 flex flex-col min-h-[88vh] justify-end">
          <h1
            className="leading-[0.95] tracking-tight mb-6"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(3.75rem, 9vw, 7.5rem)",
              color: "#ECEFEA",
              textShadow: "0 2px 24px rgba(0,0,0,0.45)",
            }}
          >
            Wander
            <br />
            <em style={{ color: "#E8D7B8" }}>deliberately.</em>
          </h1>

          <p
            className="max-w-xl text-base md:text-lg leading-relaxed mb-12"
            style={{
              color: "#C5D0C7",
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
                  className="relative pb-1.5 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[#A8E0B4] rounded-sm"
                  style={{ color: isActive ? "#A8E0B4" : "#B5C2B8" }}
                >
                  {s.name}
                  <span
                    className="absolute left-0 right-0 -bottom-0.5 h-px transition-opacity duration-300"
                    style={{
                      backgroundColor: "#A8E0B4",
                      opacity: isActive ? 1 : 0,
                    }}
                    aria-hidden
                  />
                </button>
              );
            })}
          </div>

          <div className="flex items-center gap-6 flex-wrap mb-6">
            <p className="text-sm" style={{ color: "#C5D0C7" }}>
              <span style={{ color: "#A8E0B4" }}></span> Traveling through the
              four seasons
            </p>
          </div>

          <div className="flex items-center gap-6 flex-wrap">
            <Link
              href="/sign-up"
              className="inline-flex items-center gap-2 px-6 py-3.5 rounded-full text-sm font-medium tracking-wide transition-all hover:-translate-y-0.5 active:translate-y-0"
              style={{
                backgroundColor: "#7FB68A",
                color: "#0B100D",
                boxShadow: "0 8px 24px rgba(127,182,138,0.18)",
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
