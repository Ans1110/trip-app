"use client";

import * as React from "react";
import { Clock } from "lucide-react";

import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

const HOURS = Array.from({ length: 24 }, (_, i) =>
  i.toString().padStart(2, "0"),
);
const MINUTES = ["00", "15", "30", "45"];

type TimePickerProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  className?: string;
  id?: string;
};

function splitTime(value: string): { hour: string; minute: string } {
  if (!value) return { hour: "", minute: "" };
  const [h = "", m = ""] = value.split(":");
  return { hour: h, minute: m };
}

function TimePicker({
  value,
  onChange,
  placeholder = "Pick a time",
  disabled,
  invalid,
  className,
  id,
}: TimePickerProps) {
  const [open, setOpen] = React.useState(false);
  const { hour, minute } = splitTime(value);

  const setHour = (h: string) => {
    onChange(`${h}:${minute || "00"}`);
  };
  const setMinute = (m: string) => {
    onChange(`${hour || "09"}:${m}`);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          id={id}
          disabled={disabled}
          data-invalid={invalid || undefined}
          className={cn(
            "season-transition flex w-full items-center justify-between gap-2 px-3 py-2 rounded-lg text-sm outline-none text-left",
            "bg-muted border border-border text-foreground",
            "focus:border-[color:var(--mint-ink)]",
            "disabled:opacity-60 disabled:cursor-not-allowed",
            "data-[invalid=true]:border-[rgba(220,38,38,0.4)]",
            !value && "text-subtle-foreground",
            className,
          )}
        >
          <span>{value || placeholder}</span>
          <Clock className="size-3.5 text-muted-foreground" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="p-3 w-auto">
        <div className="flex gap-3">
          <div className="flex flex-col gap-1.5">
            <p
              className="text-[10px] uppercase tracking-widest"
              style={{ color: "var(--subtle-foreground)" }}
            >
              Hour
            </p>
            <div className="grid grid-cols-6 gap-1">
              {HOURS.map((h) => {
                const active = h === hour;
                return (
                  <button
                    key={h}
                    type="button"
                    onClick={() => setHour(h)}
                    className={cn(
                      "size-7 rounded text-xs season-transition",
                      active ? "font-semibold" : "hover:bg-muted",
                    )}
                    style={
                      active
                        ? {
                            backgroundColor: "var(--season-button)",
                            color: "var(--primary-foreground)",
                          }
                        : { color: "var(--foreground)" }
                    }
                  >
                    {h}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <p
              className="text-[10px] uppercase tracking-widest"
              style={{ color: "var(--subtle-foreground)" }}
            >
              Min
            </p>
            <div className="flex flex-col gap-1">
              {MINUTES.map((m) => {
                const active = m === minute;
                return (
                  <button
                    key={m}
                    type="button"
                    onClick={() => setMinute(m)}
                    className={cn(
                      "px-3 py-1 rounded text-xs season-transition text-left",
                      active ? "font-semibold" : "hover:bg-muted",
                    )}
                    style={
                      active
                        ? {
                            backgroundColor: "var(--season-button)",
                            color: "var(--primary-foreground)",
                          }
                        : { color: "var(--foreground)" }
                    }
                  >
                    :{m}
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export { TimePicker };
