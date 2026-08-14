import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost" | "dashed";

const variants: Record<ButtonVariant, string> = {
  primary: "border-brand bg-brand text-white hover:bg-brand-hover",
  secondary: "border-line-strong bg-surface text-ink hover:border-brand hover:text-brand",
  danger: "border-danger/25 bg-danger-soft text-danger hover:border-danger/45",
  ghost: "border-transparent bg-transparent text-brand hover:bg-brand-soft",
  dashed: "border-line-strong bg-transparent text-brand hover:border-brand hover:bg-brand-soft/45 border-dashed",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  leadingIcon?: ReactNode;
  fullWidth?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button({
  variant = "primary",
  leadingIcon,
  fullWidth = false,
  className = "",
  children,
  ...props
}, ref) {
  return (
    <button
      ref={ref}
      className={`inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border px-4 py-2.5 text-sm font-bold transition-[transform,background-color,border-color,color] duration-150 active:scale-[.97] disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${fullWidth ? "w-full" : ""} ${className}`}
      {...props}
    >
      {leadingIcon}
      {children}
    </button>
  );
});
