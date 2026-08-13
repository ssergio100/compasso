import type { ElementType, HTMLAttributes, ReactNode } from "react";

interface SurfaceProps extends HTMLAttributes<HTMLElement> {
  as?: ElementType;
  children: ReactNode;
  elevated?: boolean;
}

export function Surface({ as: Component = "section", elevated = true, className = "", children, ...props }: SurfaceProps) {
  return (
    <Component
      className={`rounded-[1.375rem] border border-line bg-surface ${elevated ? "shadow-panel" : ""} ${className}`}
      {...props}
    >
      {children}
    </Component>
  );
}
