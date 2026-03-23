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
        "workspace-sidebar-shell relative flex h-full shrink-0 flex-col overflow-hidden transition-[width] duration-200",
        collapsed ? "w-[92px]" : "w-[336px]",
        className
      )}
      {...rest}
    >
      {children}
    </aside>
  );
}
