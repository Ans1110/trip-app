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
  { value: "var(--muted-foreground)", label: "Sage" },
  { value: "var(--mint)", label: "Mint" },
  { value: "var(--sky)", label: "Sky" },
  { value: "var(--pink)", label: "Sand" },
  { value: "var(--pink)", label: "Lilac" },
  { value: "var(--pink)", label: "Rose" },
  { value: "var(--pink)", label: "Peach" },
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
                      backgroundColor: isNone ? "var(--background)" : c.value,
                      borderColor: active
                        ? "var(--season-button)"
                        : isNone
                          ? "var(--muted)"
                          : "transparent",
                      boxShadow: active
                        ? "0 0 0 2px color-mix(in srgb, var(--season-button) 45%, transparent)"
                        : undefined,
                    }}
                  >
                    {isNone ? (
                      <Ban className="size-3" style={{ color: "var(--muted-foreground)" }} />
                    ) : active ? (
                      <Check
                        className="size-3.5"
                        style={{ color: "var(--primary-foreground)" }}
                      />
                    ) : null}
                  </button>
                );
              })}
            </div>
            {error ? (
              <p className="text-[11px] text-destructive">{error.message}</p>
            ) : hint ? (
              <p className="text-[11px] text-subtle-foreground">{hint}</p>
            ) : null}
          </>
        )}
      />
    </div>
  );
}

export { ControlledColorPalette };
