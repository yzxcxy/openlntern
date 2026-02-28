"use client";

import type { CSSProperties, ChangeEvent, InputHTMLAttributes } from "react";
import { Input as SemiInput } from "@douyinfe/semi-ui-19";

type UiInputProvider = "native" | "semi";

type UiInputProps = InputHTMLAttributes<HTMLInputElement> & {
  provider?: UiInputProvider;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function UiInput({
  provider = "native",
  className,
  style,
  onChange,
  ...rest
}: UiInputProps) {
  if (provider === "semi") {
    return (
      <SemiInput
        {...(rest as Record<string, unknown>)}
        onChange={(value: unknown) => {
          const nextValue =
            typeof value === "string" ? value : String(value ?? "");
          onChange?.({
            target: { value: nextValue },
          } as ChangeEvent<HTMLInputElement>);
        }}
        className={className}
        style={{ borderRadius: "var(--radius-md)", ...(style as CSSProperties) }}
      />
    );
  }

  return (
    <input
      {...rest}
      onChange={onChange}
      className={joinClasses(
        "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
        "bg-[var(--color-bg-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)]",
        "placeholder:text-[var(--color-text-muted)]",
        "outline-none transition-[border-color,box-shadow] duration-150",
        "focus:border-[var(--color-action-primary)] focus:ring-2 focus:ring-[var(--color-action-primary-soft)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
      style={style}
    />
  );
}
