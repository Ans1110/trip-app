"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { DatePicker } from "./date-picker";

type ControlledDatePickerProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  placeholder?: string;
  containerClassName?: string;
  labelClassName?: string;
  triggerClassName?: string;
  disabled?: boolean;
  min?: string;
  max?: string;
};

function ControlledDatePicker<T extends FieldValues>({
  name,
  label,
  hint,
  placeholder,
  containerClassName,
  labelClassName,
  triggerClassName,
  disabled,
  min,
  max,
}: ControlledDatePickerProps<T>) {
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
            <DatePicker
              id={name}
              value={(field.value as string | undefined) ?? ""}
              onChange={field.onChange}
              placeholder={placeholder}
              disabled={disabled}
              min={min}
              max={max}
              invalid={!!error}
              className={triggerClassName}
            />
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

export { ControlledDatePicker };
