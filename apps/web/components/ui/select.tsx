"use client";

import * as React from "react";
import { Select as RxSelect } from "radix-ui";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

const Select = RxSelect.Root;
const SelectGroup = RxSelect.Group;
const SelectValue = RxSelect.Value;

function SelectTrigger({
  className,
  children,
  ...props
}: React.ComponentProps<typeof RxSelect.Trigger>) {
  return (
    <RxSelect.Trigger
      data-slot="select-trigger"
      className={cn(
        "season-transition flex w-full items-center justify-between gap-2 px-3 py-2 rounded-lg text-sm outline-none",
        "bg-[#161E19] border border-[#1F2A24] text-[#ECEFEA]",
        "data-[placeholder]:text-[#6B7A6F]",
        "focus:border-[color:var(--season-button)]",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "aria-invalid:border-[rgba(220,38,38,0.4)]",
        className,
      )}
      {...props}
    >
      {children}
      <RxSelect.Icon asChild>
        <ChevronDown className="size-3.5 text-[#8B9A8E]" />
      </RxSelect.Icon>
    </RxSelect.Trigger>
  );
}

function SelectContent({
  className,
  position = "popper",
  sideOffset = 6,
  children,
  ...props
}: React.ComponentProps<typeof RxSelect.Content>) {
  return (
    <RxSelect.Portal>
      <RxSelect.Content
        data-slot="select-content"
        position={position}
        sideOffset={sideOffset}
        className={cn(
          "z-50 max-h-72 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-lg",
          "bg-[#121814] border border-[#1F2A24] text-[#ECEFEA] shadow-lg outline-none",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
          "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
          className,
        )}
        {...props}
      >
        <RxSelect.Viewport className="p-1">{children}</RxSelect.Viewport>
      </RxSelect.Content>
    </RxSelect.Portal>
  );
}

function SelectItem({
  className,
  children,
  ...props
}: React.ComponentProps<typeof RxSelect.Item>) {
  return (
    <RxSelect.Item
      data-slot="select-item"
      className={cn(
        "relative flex items-center gap-2 rounded-md py-1.5 pl-7 pr-2 text-sm outline-none cursor-default select-none",
        "data-[highlighted]:bg-white/5 data-[highlighted]:text-[#ECEFEA]",
        "data-[disabled]:opacity-60 data-[disabled]:cursor-not-allowed",
        className,
      )}
      {...props}
    >
      <span className="absolute left-1.5 flex size-4 items-center justify-center">
        <RxSelect.ItemIndicator>
          <Check className="size-3.5 text-[color:var(--season-button)]" />
        </RxSelect.ItemIndicator>
      </span>
      <RxSelect.ItemText>{children}</RxSelect.ItemText>
    </RxSelect.Item>
  );
}

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectItem,
};
