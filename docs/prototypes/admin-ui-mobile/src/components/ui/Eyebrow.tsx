import type { HTMLAttributes } from "react";

export function Eyebrow({ className = "", ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={`mb-1.5 text-xs font-extrabold tracking-[.1em] text-brand uppercase ${className}`}
      {...props}
    />
  );
}
