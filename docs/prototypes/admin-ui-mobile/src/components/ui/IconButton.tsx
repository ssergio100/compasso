import type { ButtonHTMLAttributes, ReactNode } from "react";

interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  children: ReactNode;
  tone?: "light" | "surface";
}

export function IconButton({ label, children, tone = "surface", className = "", ...props }: IconButtonProps) {
  const toneClass = tone === "light"
    ? "bg-white/10 text-white hover:bg-white/16"
    : "border border-line bg-surface text-ink hover:border-brand hover:text-brand";

  return (
    <button
      aria-label={label}
      className={`grid size-11 shrink-0 place-items-center rounded-xl transition-[transform,background-color,border-color,color] duration-150 active:scale-[.96] ${toneClass} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
