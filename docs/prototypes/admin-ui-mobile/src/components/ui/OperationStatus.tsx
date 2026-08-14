import { CheckCircle2, CircleAlert, Clock3, RotateCcw } from "lucide-react";
import type { ClientOperation } from "../../domain/models";
import { Button } from "./Button";

export function OperationStatus({ operation, onRetry }: { operation?: ClientOperation; onRetry?: () => void }) {
  if (!operation) return null;

  const presentation = operation.status === "applied"
    ? { Icon: CheckCircle2, title: "Aplicado", className: "bg-brand-soft text-brand" }
    : operation.status === "error"
      ? { Icon: CircleAlert, title: "Não foi possível concluir", className: "bg-danger-soft text-danger" }
      : { Icon: Clock3, title: "Aguardando sincronização", className: "bg-warning-soft text-warning" };

  return (
    <div
      className={`mt-4 flex items-start gap-3 rounded-2xl p-4 ${presentation.className}`}
      role={operation.status === "error" ? "alert" : "status"}
    >
      <presentation.Icon aria-hidden="true" className="mt-0.5 shrink-0" size={19} />
      <div className="min-w-0 flex-1">
        <strong className="block text-sm">{operation.label} · {presentation.title}</strong>
        <p className="mt-1 text-xs leading-relaxed opacity-85">{operation.detail}</p>
        {operation.status === "error" && onRetry ? (
          <Button className="mt-3" leadingIcon={<RotateCcw aria-hidden="true" size={16} />} onClick={onRetry} variant="secondary">Tentar novamente</Button>
        ) : null}
      </div>
    </div>
  );
}
