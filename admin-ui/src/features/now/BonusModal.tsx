import { AlertTriangle, Plus } from "lucide-react";
import { useState } from "react";
import { Modal } from "../../components";

export function BonusModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (minutes: number) => Promise<void> }) {
  const [value, setValue] = useState(30);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return <Modal title="Mais tempo" description="O tempo extra vale somente para hoje." onClose={onClose}><div className="preset-grid">{[15, 30, 45, 60].map((item) => <button className={value === item ? "active" : ""} key={item} onClick={() => { setValue(item); setError(""); }}><strong>{item}</strong><span>min</span></button>)}</div>{error && <div className="routine-conflict-alert" role="alert"><AlertTriangle aria-hidden="true" size={20} /><div><strong>Tempo não adicionado</strong><span>{error}</span></div></div>}<div className="modal-actions"><button onClick={onClose}>Cancelar</button><button className="primary" disabled={busy} onClick={async () => { setBusy(true); setError(""); try { await onSubmit(value); } catch (submitError) { setError(submitError instanceof Error ? submitError.message : "Não foi possível adicionar tempo."); } finally { setBusy(false); } }}><Plus size={18} />Adicionar {value}min</button></div></Modal>;
}
