"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Checkbox } from "./checkbox";

type ControlledCheckboxProps<T extends FieldValues> = {
  name: Path<T>;
  label?: React.ReactNode;
  containerClassName?: string;
  disabled?: boolean;
};

function ControlledCheckbox<T extends FieldValues>({
  name,
  label,
  containerClassName,
  disabled,
}: ControlledCheckboxProps<T>) {
  const { control } = useFormContext<T>();
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState: { error } }) => (
        <div
          className={cn(
            "flex items-center gap-2",
            containerClassName,
          )}
        >
          <Checkbox
            id={name}
            ref={field.ref}
            checked={!!field.value}
            onCheckedChange={(v) => field.onChange(v === true)}
            disabled={disabled}
            aria-invalid={!!error}
          />
          {label && (
            <label
              htmlFor={name}
              className="text-sm text-[#ECEFEA] select-none cursor-pointer"
            >
              {label}
            </label>
          )}
          {error && (
            <span className="text-[11px] text-[#FCA5A5]">{error.message}</span>
          )}
        </div>
      )}
    />
  );
}

export { ControlledCheckbox };
