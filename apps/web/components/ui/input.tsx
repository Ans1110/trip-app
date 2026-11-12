import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "season-transition w-full px-3 py-2 rounded-lg text-sm outline-none",
        "bg-muted border border-border text-foreground",
        "placeholder:text-subtle-foreground",
        "focus:border-[color:var(--mint-ink)]",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "aria-invalid:border-[rgba(220,38,38,0.4)]",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
