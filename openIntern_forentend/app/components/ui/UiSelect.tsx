"use client";

import type { SelectHTMLAttributes } from "react";

type UiSelectProvider = "native" | "semi";

type UiSelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  provider?: UiSelectProvider;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function UiSelect({
  provider = "native",
  className,
  children,
  ...rest
}: UiSelectProps) {
  // Reserved for Semi adaptation in follow-up iterations.
  if (provider === "semi") {
    return (
      <select
        {...rest}
        className={joinClasses(
          "ui-select-control",
          "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
          "bg-[rgba(255,252,247,0.92)] pl-4 pr-10 py-2 text-sm text-[var(--color-text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.58)]",
          "outline-none transition-[border-color,box-shadow,background-color] duration-150",
          "focus:border-[var(--color-action-primary)] focus:bg-[rgba(255,253,250,0.98)] focus:ring-2 focus:ring-[rgba(199,104,67,0.12)]",
          "disabled:cursor-not-allowed disabled:opacity-60",
          className
        )}
      >
        {children}
      </select>
    );
  }

  return (
    <select
      {...rest}
      className={joinClasses(
        "ui-select-control",
        "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
        "bg-[rgba(255,252,247,0.92)] pl-4 pr-14 py-3 text-sm text-[var(--color-text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.58)]",
        "outline-none transition-[border-color,box-shadow,background-color] duration-150",
        "focus:border-[var(--color-action-primary)] focus:bg-[rgba(255,253,250,0.98)] focus:ring-2 focus:ring-[rgba(199,104,67,0.12)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
    >
      {children}
    </select>
  );
}
