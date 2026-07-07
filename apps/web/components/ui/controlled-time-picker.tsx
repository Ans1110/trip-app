"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { TimePicker } from "./time-picker";

type ControlledTimePickerProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  placeholder?: string;
  containerClassName?: string;
  labelClassName?: string;
  triggerClassName?: string;
  disabled?: boolean;
};

function ControlledTimePicker<T extends FieldValues>({
  name,
  label,
  hint,
  placeholder,
  containerClassName,
  labelClassName,
  triggerClassName,
  disabled,
}: ControlledTimePickerProps<T>) {
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
            <TimePicker
              id={name}
              value={(field.value as string | undefined) ?? ""}
              onChange={field.onChange}
              placeholder={placeholder}
              disabled={disabled}
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

export { ControlledTimePicker };
