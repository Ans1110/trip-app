"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Input } from "./input";
import { Label } from "./label";

type ControlledInputProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  containerClassName?: string;
  labelClassName?: string;
  labelTrailing?: React.ReactNode;
  valueAsNumber?: boolean;
} & Omit<React.ComponentProps<"input">, "name">;

function ControlledInput<T extends FieldValues>({
  name,
  label,
  hint,
  containerClassName,
  labelClassName,
  labelTrailing,
  valueAsNumber,
  className,
  type,
  ...props
}: ControlledInputProps<T>) {
  const { control } = useFormContext<T>();
  return (
    <div className={cn("flex flex-col gap-1.5", containerClassName)}>
      {(label || labelTrailing) && (
        <div className="flex items-center justify-between">
          {label ? (
            <Label htmlFor={name} className={labelClassName}>
              {label}
            </Label>
          ) : (
            <span />
          )}
          {labelTrailing}
        </div>
      )}
      <Controller
        control={control}
        name={name}
        render={({ field, fieldState: { error } }) => {
          const value =
            field.value === undefined || field.value === null
              ? ""
              : field.value;
          return (
            <>
              <Input
                id={name}
                type={type}
                aria-invalid={!!error}
                className={className}
                {...props}
                name={field.name}
                ref={field.ref}
                value={value as string | number}
                onBlur={field.onBlur}
                onChange={(e) => {
                  if (valueAsNumber) {
                    const raw = e.target.value;
                    field.onChange(raw === "" ? undefined : Number(raw));
                  } else {
                    field.onChange(e.target.value);
                  }
                }}
              />
              {error ? (
                <p className="text-[11px] text-[#FCA5A5]">{error.message}</p>
              ) : hint ? (
                <p className="text-[11px] text-[#6B7A6F]">{hint}</p>
              ) : null}
            </>
          );
        }}
      />
    </div>
  );
}

export { ControlledInput };
