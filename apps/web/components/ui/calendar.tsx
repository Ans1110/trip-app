"use client";

import * as React from "react";
import { DayPicker } from "react-day-picker";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

type CalendarProps = React.ComponentProps<typeof DayPicker>;

function Calendar({ className, classNames, ...props }: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays
      className={cn("p-3 text-[#ECEFEA]", className)}
      classNames={{
        months: "flex flex-col gap-3",
        month: "flex flex-col gap-3",
        month_caption:
          "flex items-center justify-center h-8 text-sm font-medium",
        caption_label: "text-sm font-medium",
        nav: "flex items-center gap-1 absolute right-3 top-3",
        button_previous:
          "inline-flex items-center justify-center size-7 rounded-md hover:bg-white/5 text-[#8B9A8E] disabled:opacity-40",
        button_next:
          "inline-flex items-center justify-center size-7 rounded-md hover:bg-white/5 text-[#8B9A8E] disabled:opacity-40",
        month_grid: "w-full border-collapse",
        weekdays: "flex",
        weekday:
          "text-[#6B7A6F] text-[11px] uppercase tracking-wider w-9 h-8 flex items-center justify-center font-medium",
        week: "flex w-full",
        day: "size-9 p-0 text-center text-sm relative",
        day_button:
          "size-9 inline-flex items-center justify-center rounded-md hover:bg-white/5 outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--season-button)]/30",
        selected:
          "[&_button]:bg-[color:var(--season-button)] [&_button]:text-[#0B100D] [&_button]:hover:bg-[color:var(--season-button)]",
        today:
          "[&_button]:font-semibold [&_button]:text-[color:var(--season-button)]",
        outside: "[&_button]:text-[#3F4A43]",
        disabled: "[&_button]:opacity-40 [&_button]:pointer-events-none",
        hidden: "invisible",
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation }) =>
          orientation === "left" ? (
            <ChevronLeft className="size-4" />
          ) : (
            <ChevronRight className="size-4" />
          ),
      }}
      {...props}
    />
  );
}

export { Calendar };
