"use client";

import * as React from "react";
import { ContextMenu as RxContextMenu } from "radix-ui";

import { cn } from "@/lib/utils";

const ContextMenu = RxContextMenu.Root;
const ContextMenuTrigger = RxContextMenu.Trigger;

function ContextMenuContent({
  className,
  ...props
}: React.ComponentProps<typeof RxContextMenu.Content>) {
  return (
    <RxContextMenu.Portal>
      <RxContextMenu.Content
        data-slot="context-menu-content"
        className={cn(
          "z-[60] min-w-44 rounded-lg bg-[#121814] border border-[#1F2A24] text-[#ECEFEA] shadow-lg outline-none p-1",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
          "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
          className,
        )}
        {...props}
      />
    </RxContextMenu.Portal>
  );
}

function ContextMenuItem({
  className,
  ...props
}: React.ComponentProps<typeof RxContextMenu.Item>) {
  return (
    <RxContextMenu.Item
      className={cn(
        "relative flex items-center gap-2 rounded px-2.5 py-1.5 text-xs cursor-pointer outline-none select-none season-transition",
        "data-[highlighted]:bg-white/5",
        "data-[disabled]:opacity-50 data-[disabled]:pointer-events-none",
        className,
      )}
      {...props}
    />
  );
}

function ContextMenuSeparator({
  className,
  ...props
}: React.ComponentProps<typeof RxContextMenu.Separator>) {
  return (
    <RxContextMenu.Separator
      className={cn("my-1 h-px bg-[#1F2A24]", className)}
      {...props}
    />
  );
}

export {
  ContextMenu,
  ContextMenuTrigger,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
};
