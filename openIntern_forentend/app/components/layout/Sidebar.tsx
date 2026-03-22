import type { HTMLAttributes } from "react";

type SidebarProps = HTMLAttributes<HTMLElement> & {
  collapsed?: boolean;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function Sidebar({
  collapsed = false,
  className,
  children,
  ...rest
}: SidebarProps) {
  return (
    <aside
      className={joinClasses(
        "flex h-full flex-col",
        "bg-[var(--color-bg-surface)] transition-[width] duration-150",
        collapsed ? "w-16" : "w-72",
        className
      )}
      {...rest}
    >
      {children}
    </aside>
  );
}
