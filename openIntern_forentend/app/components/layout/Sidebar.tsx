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
        "relative flex h-full shrink-0 flex-col overflow-hidden rounded-[34px] border border-[var(--color-border-default)] bg-[var(--color-bg-sidebar)] shadow-[var(--shadow-lg)] backdrop-blur-xl transition-[width] duration-200",
        collapsed ? "w-[92px]" : "w-[336px]",
        className
      )}
      {...rest}
    >
      {children}
    </aside>
  );
}
