"use client";

import * as React from "react";
import { Popover as RxPopover } from "radix-ui";

import { cn } from "@/lib/utils";

const Popover = RxPopover.Root;
const PopoverTrigger = RxPopover.Trigger;
const PopoverAnchor = RxPopover.Anchor;

function PopoverContent({
  className,
  align = "start",
  sideOffset = 6,
  ...props
}: React.ComponentProps<typeof RxPopover.Content>) {
  return (
    <RxPopover.Portal>
      <RxPopover.Content
        data-slot="popover-content"
        align={align}
        sideOffset={sideOffset}
        className={cn(
          "z-50 rounded-lg bg-[#121814] border border-[#1F2A24] text-[#ECEFEA] shadow-lg outline-none",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
          "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
          className,
        )}
        {...props}
      />
    </RxPopover.Portal>
  );
}

export { Popover, PopoverTrigger, PopoverAnchor, PopoverContent };
