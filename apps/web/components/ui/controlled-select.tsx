"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./select";

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
  triggerClassName?: string;
  containerClassName?: string;
  labelClassName?: string;
  disabled?: boolean;
  "aria-label"?: string;
};

function ControlledSelect<T extends FieldValues>({
  name,
  label,
  hint,
  options,
  placeholder,
  triggerClassName,
  containerClassName,
  labelClassName,
  disabled,
  ...props
}: ControlledSelectProps<T>) {
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
            <Select
              value={(field.value as string | undefined) ?? ""}
              onValueChange={(v) => field.onChange(v)}
              disabled={disabled}
            >
              <SelectTrigger
                id={name}
                aria-invalid={!!error}
                className={triggerClassName}
                aria-label={props["aria-label"]}
                onBlur={field.onBlur}
                ref={field.ref}
              >
                <SelectValue placeholder={placeholder} />
              </SelectTrigger>
              <SelectContent>
                {options.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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

export { ControlledSelect };
