import type { ReactNode } from "react";

export function AppHeader({ children }: { children: ReactNode }) {
  return (
    <header className="safe-inline bg-brand-dark px-4 pt-5 pb-9 text-white">
      <div className="mx-auto flex max-w-[61.25rem] items-center justify-between gap-3">{children}</div>
    </header>
  );
}
