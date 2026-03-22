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
        "bg-[rgba(255,252,247,0.92)] px-4 py-3 text-sm text-[var(--color-text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.58)]",
        "placeholder:text-[var(--color-text-muted)]",
        "outline-none transition-[border-color,box-shadow,background-color] duration-150",
        "focus:border-[var(--color-action-primary)] focus:bg-[rgba(255,253,250,0.98)] focus:ring-2 focus:ring-[rgba(199,104,67,0.12)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
      style={style}
    />
  );
}
