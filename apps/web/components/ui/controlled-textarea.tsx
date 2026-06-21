"use client";

import {
  Controller,
  type FieldValues,
  type Path,
  useFormContext,
} from "react-hook-form";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { Textarea } from "./textarea";

type ControlledTextareaProps<T extends FieldValues> = {
  name: Path<T>;
  label?: string;
  hint?: string;
  containerClassName?: string;
  labelClassName?: string;
} & Omit<React.ComponentProps<"textarea">, "name">;

function ControlledTextarea<T extends FieldValues>({
  name,
  label,
  hint,
  containerClassName,
  labelClassName,
  className,
  ...props
}: ControlledTextareaProps<T>) {
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
            <Textarea
              id={name}
              aria-invalid={!!error}
              className={className}
              {...props}
              name={field.name}
              ref={field.ref}
              value={(field.value as string | undefined) ?? ""}
              onBlur={field.onBlur}
              onChange={(e) => field.onChange(e.target.value)}
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

export { ControlledTextarea };
