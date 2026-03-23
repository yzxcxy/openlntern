import type { ReactNode } from "react";

type AppShellProps = {
  sidebar: ReactNode;
  children: ReactNode;
};

export function AppShell({ sidebar, children }: AppShellProps) {
  return (
    <main
      className="workspace-app-shell flex h-full w-full overflow-hidden p-3 md:p-4"
      style={{ minHeight: "100dvh" }}
    >
      {sidebar}
      <section className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <div id="main-content" className="workspace-main-content flex min-h-0 flex-1 overflow-hidden">
          {children}
        </div>
      </section>
    </main>
  );
}
