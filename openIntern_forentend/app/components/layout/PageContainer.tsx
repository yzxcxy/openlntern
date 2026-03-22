import type { HTMLAttributes } from "react";

type PageContainerProps = HTMLAttributes<HTMLDivElement>;

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function PageContainer({
  className,
  children,
  ...rest
}: PageContainerProps) {
  return (
    <div
      className={joinClasses(
        "relative mx-auto h-full w-full max-w-[1720px]",
        className
      )}
      {...rest}
    >
      {children}
    </div>
  );
}
