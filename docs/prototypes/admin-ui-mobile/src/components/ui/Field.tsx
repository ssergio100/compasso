import { forwardRef, type InputHTMLAttributes } from "react";

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  help?: string;
  error?: string;
}

export const Field = forwardRef<HTMLInputElement, FieldProps>(function Field(
  { label, help, error, id, className = "", ...props },
  ref,
) {
  const inputId = id ?? props.name;
  const helpId = help && inputId ? `${inputId}-help` : undefined;
  const errorId = error && inputId ? `${inputId}-error` : undefined;

  return (
    <div className="grid gap-2">
      <label className="text-sm font-bold text-ink" htmlFor={inputId}>{label}</label>
      <input
        ref={ref}
        id={inputId}
        aria-describedby={errorId ?? helpId}
        aria-invalid={error ? true : undefined}
        className={`min-h-11 w-full rounded-xl border bg-surface px-3.5 py-2.5 font-normal text-ink transition-colors placeholder:text-muted/80 ${error ? "border-danger hover:border-danger focus:border-danger" : "border-line-strong hover:border-brand focus:border-brand"} ${className}`}
        {...props}
      />
      {error ? <span id={errorId} className="text-xs font-bold leading-relaxed text-danger">{error}</span> : null}
      {!error && help ? <span id={helpId} className="text-xs font-normal leading-relaxed text-muted">{help}</span> : null}
    </div>
  );
});
