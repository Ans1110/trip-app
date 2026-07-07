"use client";

import { Check, Ban } from "lucide-react";
import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";

export const CALENDAR_COLORS = [
  { value: "", label: "None" },
  { value: "#8B9A8E", label: "Sage" },
  { value: "#A8E0B4", label: "Mint" },
  { value: "#7FB3D5", label: "Sky" },
  { value: "#E8D7B8", label: "Sand" },
  { value: "#C3A6E0", label: "Lilac" },
  { value: "#F5A6A6", label: "Rose" },
  { value: "#F5C89F", label: "Peach" },
];

type ControlledColorPaletteProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  containerClassName?: string;
  labelClassName?: string;
  disabled?: boolean;
};

function ControlledColorPalette<T extends FieldValues>({
  name,
  label,
  hint,
  containerClassName,
  labelClassName,
  disabled,
}: ControlledColorPaletteProps<T>) {
  const { control } = useFormContext<T>();
  return (
    <div className={cn("flex flex-col gap-1.5", containerClassName)}>
      {label && (
        <Label htmlFor={name} className={labelClassName}>
          {label}
        </Label>
      )}
      <Controller
        control={control}
        name={name}
        render={({ field, fieldState: { error } }) => (
          <>
            <div id={name} className="flex flex-wrap gap-2">
              {CALENDAR_COLORS.map((c) => {
                const active = (field.value as string) === c.value;
                const isNone = c.value === "";
                return (
                  <button
                    key={c.value || "none"}
                    type="button"
                    aria-label={c.label}
                    aria-pressed={active}
                    title={c.label}
                    disabled={disabled}
                    onClick={() => field.onChange(c.value)}
                    className={cn(
                      "size-7 rounded-full flex items-center justify-center season-transition",
                      "border hover:scale-110",
                      disabled &&
                        "opacity-50 cursor-not-allowed hover:scale-100",
                    )}
                    style={{
                      backgroundColor: isNone ? "#0B100D" : c.value,
                      borderColor: active
                        ? "var(--season-button)"
                        : isNone
                          ? "#1F2A24"
                          : "transparent",
                      boxShadow: active
                        ? "0 0 0 2px color-mix(in srgb, var(--season-button) 45%, transparent)"
                        : undefined,
                    }}
                  >
                    {isNone ? (
                      <Ban className="size-3" style={{ color: "#8B9A8E" }} />
                    ) : active ? (
                      <Check
                        className="size-3.5"
                        style={{ color: "#0B100D" }}
                      />
                    ) : null}
                  </button>
                );
              })}
            </div>
            {error ? (
              <p className="text-[11px] text-[#FCA5A5]">{error.message}</p>
            ) : hint ? (
              <p className="text-[11px] text-[#6B7A6F]">{hint}</p>
            ) : null}
          </>
        )}
      />
    </div>
  );
}

export { ControlledColorPalette };
