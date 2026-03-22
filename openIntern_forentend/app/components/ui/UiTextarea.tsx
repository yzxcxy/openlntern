"use client";

import type { TextareaHTMLAttributes } from "react";

type UiTextareaProvider = "native" | "semi";

type UiTextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  provider?: UiTextareaProvider;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function UiTextarea({
  provider = "native",
  className,
  ...rest
}: UiTextareaProps) {
  // Reserved for Semi adaptation in follow-up iterations.
  if (provider === "semi") {
    return (
      <textarea
        {...rest}
        className={joinClasses(
          "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
          "bg-[rgba(255,252,247,0.92)] px-4 py-3 text-sm text-[var(--color-text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.58)]",
          "placeholder:text-[var(--color-text-muted)]",
          "outline-none transition-[border-color,box-shadow,background-color] duration-150",
          "focus:border-[var(--color-action-primary)] focus:bg-[rgba(255,253,250,0.98)] focus:ring-2 focus:ring-[rgba(199,104,67,0.12)]",
          "disabled:cursor-not-allowed disabled:opacity-60",
          className
        )}
      />
    );
  }

  return (
    <textarea
      {...rest}
      className={joinClasses(
        "block w-full rounded-[var(--radius-md)] border border-[var(--color-border-default)]",
        "bg-[rgba(255,252,247,0.92)] px-4 py-3 text-sm text-[var(--color-text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.58)]",
        "placeholder:text-[var(--color-text-muted)]",
        "outline-none transition-[border-color,box-shadow,background-color] duration-150",
        "focus:border-[var(--color-action-primary)] focus:bg-[rgba(255,253,250,0.98)] focus:ring-2 focus:ring-[rgba(199,104,67,0.12)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
    />
  );
}
