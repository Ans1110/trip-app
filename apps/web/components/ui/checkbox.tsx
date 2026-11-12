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
        "bg-muted border border-border",
        "focus-visible:border-[color:var(--mint-ink)] focus-visible:ring-2 focus-visible:ring-[color:var(--mint-ink)]/30",
        "data-[state=checked]:bg-[color:var(--mint-ink)] data-[state=checked]:border-[color:var(--mint-ink)]",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "aria-invalid:border-[rgba(220,38,38,0.4)]",
        className,
      )}
      {...props}
    >
      <RxCheckbox.Indicator className="flex items-center justify-center text-primary-foreground">
        <Check className="size-3" strokeWidth={3} />
      </RxCheckbox.Indicator>
    </RxCheckbox.Root>
  );
}

export { Checkbox };
