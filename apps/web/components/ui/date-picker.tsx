"use client";

import * as React from "react";
import { format, parse, isValid } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { Calendar } from "./calendar";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

export const ISO_DATE_FORMAT = "yyyy-MM-dd";

const parseISO = (value: string): Date | undefined => {
  if (!value) return undefined;
  const d = parse(value, ISO_DATE_FORMAT, new Date());
  return isValid(d) ? d : undefined;
};

type DatePickerProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  min?: string;
  max?: string;
  invalid?: boolean;
  id?: string;
};

function DatePicker({
  value,
  onChange,
  placeholder = "Pick a date",
  className,
  disabled,
  min,
  max,
  invalid,
  id,
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false);
  const selected = parseISO(value);
  const minDate = min ? parseISO(min) : undefined;
  const maxDate = max ? parseISO(max) : undefined;

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
            "bg-[#161E19] border border-[#1F2A24] text-[#ECEFEA]",
            "focus:border-[color:var(--season-button)]",
            "disabled:opacity-60 disabled:cursor-not-allowed",
            "data-[invalid=true]:border-[rgba(220,38,38,0.4)]",
            !selected && "text-[#6B7A6F]",
            className,
          )}
        >
          <span>{selected ? format(selected, "MMM d, yyyy") : placeholder}</span>
          <CalendarIcon className="size-3.5 text-[#8B9A8E]" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="p-0 w-auto">
        <Calendar
          mode="single"
          selected={selected}
          onSelect={(d) => {
            onChange(d ? format(d, ISO_DATE_FORMAT) : "");
            if (d) setOpen(false);
          }}
          disabled={
            minDate || maxDate
              ? [
                  ...(minDate ? [{ before: minDate }] : []),
                  ...(maxDate ? [{ after: maxDate }] : []),
                ]
              : undefined
          }
          defaultMonth={selected ?? minDate}
          autoFocus
        />
      </PopoverContent>
    </Popover>
  );
}

export { DatePicker };
