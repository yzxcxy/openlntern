"use client";

import type { ButtonHTMLAttributes, CSSProperties } from "react";
import { Button as SemiButton } from "@douyinfe/semi-ui-19";

type UiButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type UiButtonSize = "sm" | "md" | "lg";
type UiButtonProvider = "native" | "semi";

type UiButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "size"> & {
  provider?: UiButtonProvider;
  variant?: UiButtonVariant;
  size?: UiButtonSize;
  loading?: boolean;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

const SIZE_CLASSES: Record<UiButtonSize, string> = {
  sm: "h-8 px-3 text-xs",
  md: "h-10 px-4 text-sm",
  lg: "h-11 px-5 text-sm",
};

const NATIVE_VARIANT_CLASSES: Record<UiButtonVariant, string> = {
  primary:
    "bg-[var(--color-action-primary)] text-white hover:bg-[var(--color-action-primary-hover)] active:bg-[var(--color-action-primary-active)]",
  secondary:
    "bg-[var(--color-bg-surface)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] hover:bg-[var(--color-bg-page)]",
  ghost:
    "bg-transparent text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)]",
  danger:
    "bg-[var(--color-state-error)] text-white hover:opacity-95 active:opacity-90",
};

export function UiButton({
  provider = "native",
  variant = "primary",
  size = "md",
  className,
  style,
  children,
  type = "button",
  disabled,
  loading = false,
  ...rest
}: UiButtonProps) {
  if (provider === "semi") {
    const semiType =
      variant === "danger"
        ? "danger"
        : variant === "primary"
          ? "primary"
          : "tertiary";
    const semiTheme =
      variant === "ghost"
        ? "borderless"
        : variant === "primary" || variant === "danger"
          ? "solid"
          : "light";
    return (
      <SemiButton
        {...(rest as Record<string, unknown>)}
        htmlType={type}
        loading={loading}
        disabled={disabled || loading}
        type={semiType as "danger" | "primary" | "tertiary"}
        theme={semiTheme as "borderless" | "light" | "solid"}
        className={className}
        style={{ borderRadius: "var(--radius-md)", ...(style as CSSProperties) }}
      >
        {children}
      </SemiButton>
    );
  }

  return (
    <button
      type={type}
      className={joinClasses(
        "inline-flex items-center justify-center gap-2 rounded-[var(--radius-md)] font-semibold transition-[background-color,border-color,box-shadow,opacity] duration-150 outline-none",
        "focus-visible:ring-2 focus-visible:ring-[var(--color-action-primary-soft)] focus-visible:ring-offset-2",
        "disabled:pointer-events-none disabled:opacity-60",
        SIZE_CLASSES[size],
        NATIVE_VARIANT_CLASSES[variant],
        className
      )}
      style={style}
      disabled={disabled || loading}
      {...rest}
    >
      {loading && (
        <span
          className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"
          aria-hidden="true"
        />
      )}
      {children}
    </button>
  );
}
