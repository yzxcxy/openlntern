import type { ReactNode } from "react";

type AppShellProps = {
  sidebar: ReactNode;
  children: ReactNode;
};

export function AppShell({ sidebar, children }: AppShellProps) {
  return (
    <main className="flex h-full w-screen overflow-hidden" style={{ minHeight: "100dvh" }}>
      {sidebar}
      <section className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="flex-1 overflow-hidden rounded-l-[28px] bg-[var(--color-bg-page)]">
          {children}
        </div>
      </section>
    </main>
  );
}
