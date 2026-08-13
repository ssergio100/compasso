import { AlertTriangle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "./Button";
import styles from "./ConfirmDialog.module.css";

interface ConfirmDialogProps {
  confirmLabel: string;
  description: string;
  open: boolean;
  title: string;
  onClose: () => void;
  onConfirm: () => boolean | void | Promise<boolean | void>;
}

export function ConfirmDialog({
  confirmLabel,
  description,
  open,
  title,
  onClose,
  onConfirm,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const submittingRef = useRef(false);
  const [submitting, setSubmitting] = useState(false);
  const titleId = "confirm-dialog-title";

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      if (typeof dialog.showModal === "function") dialog.showModal();
      else dialog.setAttribute("open", "");
      window.setTimeout(() => cancelRef.current?.focus(), 0);
    } else if (!open && dialog.open) {
      if (typeof dialog.close === "function") dialog.close();
      else dialog.removeAttribute("open");
    }
  }, [open]);

  return (
    <dialog
      aria-labelledby={titleId}
      className={styles.dialog}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      ref={dialogRef}
    >
      <div className="p-5 sm:p-6">
        <span className="grid size-11 place-items-center rounded-xl bg-danger-soft text-danger">
          <AlertTriangle aria-hidden="true" size={22} />
        </span>
        <h2 className="mt-4 text-xl font-extrabold" id={titleId}>{title}</h2>
        <p className="mt-2 text-sm leading-relaxed text-muted">{description}</p>
        <div className="mt-6 grid grid-cols-2 gap-2.5">
          <Button disabled={submitting} onClick={onClose} ref={cancelRef} type="button" variant="secondary">Cancelar</Button>
          <Button
            disabled={submitting}
            onClick={async () => {
              if (submittingRef.current) return;
              submittingRef.current = true;
              setSubmitting(true);
              try {
                const confirmed = await onConfirm();
                if (confirmed !== false) onClose();
              } catch {
                // The caller owns the visible error message; keep the dialog open.
              } finally {
                submittingRef.current = false;
                setSubmitting(false);
              }
            }}
            type="button"
            variant="danger"
          >
            {submitting ? "Processando…" : confirmLabel}
          </Button>
        </div>
      </div>
    </dialog>
  );
}
