import * as React from "react";

import { cn } from "@/lib/utils";

function Select({ className, children, ...props }: React.ComponentProps<"select">) {
  return (
    <select
      data-slot="select"
      className={cn(
        "season-transition w-full px-3 py-2 rounded-lg text-sm outline-none appearance-none",
        "bg-[#161E19] border border-[#1F2A24] text-[#ECEFEA]",
        "focus:border-[color:var(--season-button)]",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "aria-invalid:border-[rgba(220,38,38,0.4)]",
        className,
      )}
      {...props}
    >
      {children}
    </select>
  );
}

export { Select };
