"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { Select } from "./select";

export type SelectOption = {
  value: string;
  label: string;
};

type ControlledSelectProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  options: SelectOption[];
  placeholder?: string;
  containerClassName?: string;
} & Omit<React.ComponentProps<"select">, "name" | "children">;

function ControlledSelect<T extends FieldValues>({
  name,
  label,
  hint,
  options,
  placeholder,
  containerClassName,
  className,
  ...props
}: ControlledSelectProps<T>) {
  const { control } = useFormContext<T>();
  return (
    <div className={cn("flex flex-col gap-1.5", containerClassName)}>
      {label && <Label htmlFor={name}>{label}</Label>}
      <Controller
        control={control}
        name={name}
        render={({ field, fieldState: { error } }) => (
          <>
            <Select
              id={name}
              aria-invalid={!!error}
              className={className}
              {...props}
              name={field.name}
              ref={field.ref}
              value={(field.value as string | undefined) ?? ""}
              onBlur={field.onBlur}
              onChange={(e) => field.onChange(e.target.value)}
            >
              {placeholder && (
                <option value="" disabled>
                  {placeholder}
                </option>
              )}
              {options.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
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

export { ControlledSelect };
