"use client";

import * as React from "react";
import { Checkbox as RxCheckbox } from "radix-ui";
import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

function Checkbox({
  className,
  ...props
}: React.ComponentProps<typeof RxCheckbox.Root>) {
  return (
    <RxCheckbox.Root
      data-slot="checkbox"
      className={cn(
        "season-transition inline-flex size-4 shrink-0 items-center justify-center rounded-[4px] outline-none",
        "bg-[#161E19] border border-[#1F2A24]",
        "focus-visible:border-[color:var(--season-button)] focus-visible:ring-2 focus-visible:ring-[color:var(--season-button)]/30",
        "data-[state=checked]:bg-[color:var(--season-button)] data-[state=checked]:border-[color:var(--season-button)]",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "aria-invalid:border-[rgba(220,38,38,0.4)]",
        className,
      )}
      {...props}
    >
      <RxCheckbox.Indicator className="flex items-center justify-center text-[#0B100D]">
        <Check className="size-3" strokeWidth={3} />
      </RxCheckbox.Indicator>
    </RxCheckbox.Root>
  );
}

export { Checkbox };
