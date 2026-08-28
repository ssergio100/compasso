import { Plus } from "lucide-react";
import { useState } from "react";
import { Modal } from "../../components";
import type { AvatarKey } from "../../types";
import { AvatarPicker } from "../../visuals";

export function DeviceModal({ onClose, onSubmit }: {
  onClose: () => void;
  onSubmit: (name: string, avatarKey: AvatarKey) => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [avatarKey, setAvatarKey] = useState<AvatarKey>("cat");
  const [busy, setBusy] = useState(false);
  return <Modal title="Novo computador" description="Dê um nome e uma identidade antes de parear o agente." onClose={onClose}><form className="modal-form" onSubmit={async (event) => { event.preventDefault(); setBusy(true); try { await onSubmit(name.trim(), avatarKey); } finally { setBusy(false); } }}><label>Nome<input autoFocus placeholder="Ex.: Notebook da Ana" value={name} onChange={(event) => setName(event.target.value)} /></label><AvatarPicker value={avatarKey} onChange={setAvatarKey} /><div className="modal-actions"><button type="button" onClick={onClose}>Cancelar</button><button className="primary" disabled={!name.trim() || busy}><Plus size={18} />Adicionar</button></div></form></Modal>;
}
