import { X } from "lucide-react";
import { useAppState } from "../../app/AppState";

export function Toast() {
  const { toast, dismissToast } = useAppState();
  if (!toast) return null;

  return (
    <div className="fixed right-4 bottom-24 left-4 z-50 mx-auto flex max-w-md items-center justify-between gap-3 rounded-xl bg-brand-dark px-4 py-3 text-sm font-medium text-white shadow-panel" role="status">
      <span>{toast}</span>
      <button aria-label="Fechar aviso" className="grid size-8 shrink-0 place-items-center rounded-lg hover:bg-white/10" onClick={dismissToast} type="button">
        <X aria-hidden="true" size={17} />
      </button>
    </div>
  );
}
