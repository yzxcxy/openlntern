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
          "bg-[var(--color-bg-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)]",
          "placeholder:text-[var(--color-text-muted)]",
          "outline-none transition-[border-color,box-shadow] duration-150",
          "focus:border-[var(--color-action-primary)] focus:ring-2 focus:ring-[var(--color-action-primary-soft)]",
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
        "bg-[var(--color-bg-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)]",
        "placeholder:text-[var(--color-text-muted)]",
        "outline-none transition-[border-color,box-shadow] duration-150",
        "focus:border-[var(--color-action-primary)] focus:ring-2 focus:ring-[var(--color-action-primary-soft)]",
        "disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
    />
  );
}
