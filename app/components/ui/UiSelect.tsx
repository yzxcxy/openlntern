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
          "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
          "bg-[var(--color-bg-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)]",
          "outline-none transition-[border-color,box-shadow] duration-150",
          "focus:border-[var(--color-action-primary)] focus:ring-2 focus:ring-[var(--color-action-primary-soft)]",
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
        "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
        "bg-[var(--color-bg-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)]",
        "outline-none transition-[border-color,box-shadow] duration-150",
        "focus:border-[var(--color-action-primary)] focus:ring-2 focus:ring-[var(--color-action-primary-soft)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
    >
      {children}
    </select>
  );
}
