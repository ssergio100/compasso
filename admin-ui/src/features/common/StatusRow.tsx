import type { LucideIcon } from "lucide-react";

function statusTone(value: string) {
  if (["Conectado", "Ativa", "Ativo", "Em andamento", "Configurada", "Liberado"].includes(value)) return "state-positive";
  if (["Pausado", "Aguardando", "Pausando…", "Retomando…", "Bloqueando…", "Desbloqueando…"].includes(value)) return "state-warning";
  if (["Desconectado", "Offline", "Inativo", "Não configurada", "Bloqueado"].includes(value)) return "state-negative";
  return "";
}

export function StatusRow({ Icon, label, value }: { Icon: LucideIcon; label: string; value: string }) {
  return <div className="info-row"><Icon size={20} /><span>{label}</span><strong className={statusTone(value)}>{value}</strong></div>;
}
