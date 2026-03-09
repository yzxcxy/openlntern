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
        "mx-auto h-full w-full max-w-[1440px] px-4 py-4 md:px-6",
        className
      )}
      {...rest}
    >
      {children}
    </div>
  );
}
