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
  sm: "h-9 px-3.5 text-xs",
  md: "h-11 px-5 text-sm",
  lg: "h-12 px-6 text-sm",
};

const NATIVE_VARIANT_CLASSES: Record<UiButtonVariant, string> = {
  primary: "ui-button-primary border",
  secondary: "ui-button-secondary border",
  ghost: "ui-button-ghost border",
  danger: "ui-button-danger border",
};

const SEMI_VARIANT_CLASSES: Record<UiButtonVariant, string> = {
  primary: "ui-semi-button ui-semi-button-primary",
  secondary: "ui-semi-button ui-semi-button-secondary",
  ghost: "ui-semi-button ui-semi-button-ghost",
  danger: "ui-semi-button ui-semi-button-danger",
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
        className={joinClasses(
          SIZE_CLASSES[size],
          SEMI_VARIANT_CLASSES[variant],
          className
        )}
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
        "inline-flex items-center justify-center gap-2 rounded-[var(--radius-md)] font-semibold tracking-[-0.01em] transition-[background-color,border-color,box-shadow,opacity,color,transform,filter] duration-150 outline-none",
        "focus-visible:ring-2 focus-visible:ring-[rgba(199,104,67,0.18)] focus-visible:ring-offset-2 focus-visible:ring-offset-[rgba(255,249,242,0.92)]",
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
