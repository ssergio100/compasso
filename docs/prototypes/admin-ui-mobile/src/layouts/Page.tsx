import type { ReactNode } from "react";

export function Page({ children, narrow = false }: { children: ReactNode; narrow?: boolean }) {
  return (
    <main id="main-content" className={`relative mx-auto -mt-5 w-full px-4 pb-8 ${narrow ? "max-w-[45rem]" : "max-w-[63.25rem]"}`}>
      {children}
    </main>
  );
}
