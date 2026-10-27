import React from "react";

export interface AppShellProps {
  sidebar: React.ReactNode;
  topbar: React.ReactNode;
  tabs?: React.ReactNode;
  children: React.ReactNode;
}

export function AppShell({ sidebar, topbar, tabs, children }: AppShellProps) {
  return (
    <div className="app-shell flex h-screen overflow-hidden bg-white text-text-body">
      {sidebar}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden bg-white">
        {topbar}
        {tabs}
        <main className="flex-1 overflow-y-auto bg-white px-6 py-8 md:px-10" id="viewport-main">
          <div className="mx-auto max-w-[1500px]">{children}</div>
        </main>
      </div>
    </div>
  );
}
