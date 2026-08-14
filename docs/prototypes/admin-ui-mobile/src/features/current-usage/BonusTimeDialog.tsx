import { Clock3, Plus, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "../../components/ui/Button";
import { IconButton } from "../../components/ui/IconButton";
import { formatClock, formatMinutes } from "../../lib/formatters";
import styles from "./BonusTimeDialog.module.css";

interface BonusTimeDialogProps {
  clientName: string;
  open: boolean;
  onClose: () => void;
  onConfirm: (minutes: number) => void | Promise<void>;
}

const presets = [15, 30, 60];

export function BonusTimeDialog({ clientName, open, onClose, onConfirm }: BonusTimeDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [minutes, setMinutes] = useState(30);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      if (typeof dialog.showModal === "function") dialog.showModal();
      else dialog.setAttribute("open", "");
      window.setTimeout(() => dialog.querySelector<HTMLInputElement>("input")?.focus(), 0);
    } else if (!open && dialog.open) {
      if (typeof dialog.close === "function") dialog.close();
      else dialog.removeAttribute("open");
    }
  }, [open]);

  return (
    <dialog
      aria-labelledby="bonus-time-title"
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
        <header className="flex items-start justify-between gap-4">
          <div>
            <span className="mb-2 inline-flex items-center gap-1.5 text-xs font-extrabold uppercase tracking-[.08em] text-brand">
              <Clock3 aria-hidden="true" size={15} /> Ajuste rápido
            </span>
            <h2 className="text-xl font-extrabold" id="bonus-time-title">Adicionar tempo</h2>
            <p className="mt-1 text-sm leading-relaxed text-muted">Libere mais tempo para {clientName}.</p>
          </div>
          <IconButton label="Fechar" onClick={onClose} type="button"><X aria-hidden="true" size={20} /></IconButton>
        </header>

        <div className="py-6 text-center">
          <div aria-live="polite" className={styles.clock}>
            <strong className="numbers-tabular text-2xl tracking-[-.04em]">+{formatClock(minutes)}</strong>
          </div>
        </div>

        <label className="block text-sm font-bold" htmlFor="bonus-time-range">
          Tempo adicional
          <span className="float-right text-brand">{formatMinutes(minutes)}</span>
        </label>
        <input
          aria-valuetext={formatMinutes(minutes)}
          className={`${styles.range} mt-4`}
          id="bonus-time-range"
          max="240"
          min="15"
          onChange={(event) => setMinutes(Number(event.target.value))}
          step="15"
          type="range"
          value={minutes}
        />
        <div className="mt-5 grid grid-cols-3 gap-2" aria-label="Atalhos de tempo">
          {presets.map((preset) => (
            <button
              aria-pressed={minutes === preset}
              className={`min-h-11 rounded-xl border px-3 text-sm font-extrabold transition-[transform,background-color,border-color,color] active:scale-[.97] ${minutes === preset ? "border-brand bg-brand-soft text-brand" : "border-line text-ink hover:border-brand/45"}`}
              key={preset}
              onClick={() => setMinutes(preset)}
              type="button"
            >
              {formatMinutes(preset)}
            </button>
          ))}
        </div>

        <footer className="mt-6 grid grid-cols-2 gap-2.5">
          <Button onClick={onClose} type="button" variant="secondary">Cancelar</Button>
          <Button
            disabled={submitting}
            leadingIcon={<Plus aria-hidden="true" size={18} />}
            onClick={async () => {
              setSubmitting(true);
              try {
                await onConfirm(minutes);
                onClose();
              } finally {
                setSubmitting(false);
              }
            }}
            type="button"
          >
            {submitting ? "Adicionando…" : `Adicionar ${formatMinutes(minutes)}`}
          </Button>
        </footer>
      </div>
    </dialog>
  );
}
